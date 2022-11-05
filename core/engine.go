// Copyright (c) 2022 The rcproxy Authors
// Copyright (c) 2019 Andy Pan
// Copyright (c) 2018 Joshua J Baker
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build linux || freebsd || dragonfly || darwin
// +build linux freebsd dragonfly darwin

package core

import (
	"net"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	perrors "github.com/pkg/errors"
	"golang.org/x/sys/unix"

	"rcproxy/core/internal/netpoll"
	"rcproxy/core/internal/socket"
	"rcproxy/core/pkg/errors"
	"rcproxy/core/pkg/logging"
)

type engine struct {
	ln           *listener      // the listener for accepting new connections
	el           *eventloop     // event-loops
	wg           sync.WaitGroup // event-loop close WaitGroup
	opts         *Options       // options with engine
	once         sync.Once      // make sure only signalShutdown once
	cond         *sync.Cond     // shutdown signaler
	eventHandler EventHandler   // user eventHandler
	inShutdown   int32          // whether the engine is in shutdown
}

func (eng *engine) isInShutdown() bool {
	return atomic.LoadInt32(&eng.inShutdown) == 1
}

// waitForShutdown waits for a signal to shut down.
func (eng *engine) waitForShutdown() {
	eng.cond.L.Lock()
	eng.cond.Wait()
	eng.cond.L.Unlock()
}

// signalShutdown signals the engine to shut down.
func (eng *engine) signalShutdown() {
	eng.once.Do(func() {
		eng.cond.L.Lock()
		eng.cond.Signal()
		eng.cond.L.Unlock()
	})
}

func (eng *engine) startEventLoop() {
	eng.wg.Add(1)
	go func() {
		eng.el.run()
		eng.wg.Done()
	}()
}

func (eng *engine) closeEventLoops() {
	_ = eng.el.poller.Close()
}

func (eng *engine) start() (err error) {
	ln := eng.ln
	eng.ln = nil
	var p *netpoll.Poller
	if p, err = netpoll.OpenPoller(); err == nil {
		el := new(eventloop)
		el.ln = ln
		el.engine = eng
		el.poller = p
		el.buffer = make([]byte, eng.opts.ReadBufferCap)
		el.connections = make(map[int]*conn)
		el.eventHandler = eng.eventHandler
		if err = el.poller.AddRead(el.ln.packPollAttachment(el.accept)); err != nil {
			return
		}
		eng.el = el
	} else {
		return
	}

	if eng.opts.RedisPreconnect {
		// Initialize connection to the back-end redis cluster
		for _, pool := range EngineGlobal.ProxyPool {
			if pool.Get() == nil {
				return perrors.Wrapf(err, "redis preconnect failed, addr: %s", pool.Addr)
			}
		}
	}

	// Start event-loop in background.
	eng.startEventLoop()
	return
}

func (eng *engine) stop(s Engine) {
	// Wait on a signal for shutdown
	eng.waitForShutdown()

	eng.eventHandler.OnShutdown(s)

	err := eng.el.poller.UrgentTrigger(func(_ interface{}) error { return errors.ErrEngineShutdown }, nil)
	if err != nil {
		logging.Errorf("failed to call UrgentTrigger on sub event-loop when stopping engine: %v", err)
	}

	// Wait on all loops to complete reading events
	eng.wg.Wait()

	eng.closeEventLoops()

	atomic.StoreInt32(&eng.inShutdown, 1)
}

// Dial establishing a connection with redis
func (eng *engine) Dial(address string, isSlave bool) (SConn, error) {
	c, err := net.DialTimeout("tcp", address, time.Duration(eng.opts.RedisConnectionTimeout)*time.Millisecond)
	if err != nil {
		GlobalStats.RedisServerCreateConnError.WithLabelValues(address).Inc()
		logging.Errorf("failed to dial redis %s, error: %s", address, err)
		return nil, err
	}

	defer c.Close()

	sc, ok := c.(syscall.Conn)
	if !ok {
		return nil, perrors.New("failed to convert net.Conn to syscall.Conn")
	}
	rc, err := sc.SyscallConn()
	if err != nil {
		return nil, perrors.New("failed to get syscall.RawConn from net.Conn")
	}

	var DupFD int
	e := rc.Control(func(fd uintptr) {
		DupFD, err = unix.Dup(int(fd))
	})
	if err != nil {
		return nil, err
	}
	if e != nil {
		return nil, e
	}

	if err = socket.SetNoDelay(DupFD, 1); err != nil {
		return nil, err
	}

	if err = os.NewSyscallError("fcntl nonblock", unix.SetNonblock(DupFD, true)); err != nil {
		return nil, err
	}

	if eng.opts.TCPKeepAlive > 0 {
		if err = socket.SetKeepAlivePeriod(DupFD, int(eng.opts.TCPKeepAlive/time.Second)); err != nil {
			return nil, err
		}
	}

	if eng.opts.SocketSendBuffer > 0 {
		if err = socket.SetSendBuffer(DupFD, eng.opts.SocketSendBuffer); err != nil {
			return nil, err
		}
	}
	if eng.opts.SocketRecvBuffer > 0 {
		if err = socket.SetRecvBuffer(DupFD, eng.opts.SocketRecvBuffer); err != nil {
			return nil, err
		}
	}

	var initStatus InitializeStatus
	if len(eng.opts.RedisPasswd) > 0 {
		initStatus = InitializeNone
	} else {
		initStatus = Initialized
	}

	var (
		gc SConn
	)
	switch c.(type) {
	case *net.TCPConn:
		gc = newTCPConn(DupFD, eng.el, c.LocalAddr(), c.RemoteAddr(), ConnServer, initStatus, isSlave)
	default:
		return nil, errors.ErrUnsupportedProtocol
	}

	if err := eng.el.poller.AddRead(gc.(*conn).pollAttachment); err != nil {
		_ = unix.Close(gc.(*conn).fd)
		gc.(*conn).releaseTCP()
		return nil, err
	}
	eng.el.connections[gc.(*conn).fd] = gc.(*conn)

	if err := eng.el.open(gc.(*conn)); err != nil {
		return nil, err
	}
	return gc, nil
}

func serve(eventHandler EventHandler, listener *listener, options *Options, protoAddr string) error {
	eng := new(engine)
	eng.opts = options
	eng.eventHandler = eventHandler
	eng.ln = listener

	eng.cond = sync.NewCond(&sync.Mutex{})

	e := Engine{
		eng:         eng,
		ProxyPool:   make(map[string]*Pool),
		cCodec:      CRespCodec{options.RedisMsgMaxLength},
		sCodec:      SRespCodec{options.RedisMsgMaxLength},
		clusterChan: make(chan []byte, 3),
		ClusterNodes: ClusterNodes{
			redisAddrs:   options.RedisServers,
			passwd:       options.RedisPasswd,
			redisWrapper: new(redisWrapper),
		},
	}

	serverList := strings.Split(options.RedisServers, ",")
	if len(serverList) < 1 {
		logging.Errorf("redis addr not found from conf.redis.servers")
		return nil
	}

	switch eng.eventHandler.OnBoot(e) {
	case None:
	case Shutdown:
		return nil
	}

	for _, addr := range serverList {
		e.ProxyPool[addr] = eng.newPool(addr, false)
		e.ProxyAddrs = append(e.ProxyAddrs, addr)
	}
	EngineGlobal = &e
	go EngineGlobal.ClusterNodes.loopClusterNodes()
	go statsLoop()

	if err := eng.start(); err != nil {
		eng.closeEventLoops()
		logging.Errorf("gnet engine is stopping with error: %v", err)
		return err
	}
	defer eng.stop(e)

	allEngines.Store(protoAddr, eng)

	return nil
}
