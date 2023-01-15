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
	"bytes"
	"errors"
	"fmt"
	"os"
	"sync/atomic"
	"time"

	"golang.org/x/sys/unix"

	"rcproxy/core/codec"
	"rcproxy/core/internal/io"
	"rcproxy/core/internal/netpoll"
	gerrors "rcproxy/core/pkg/errors"
	"rcproxy/core/pkg/logging"
)

type eventloop struct {
	ln           *listener       // listener
	idx          int             // loop index in the engine loops list
	cache        bytes.Buffer    // temporary buffer for scattered bytes
	engine       *engine         // engine in loop
	poller       *netpoll.Poller // epoll or kqueue
	buffer       []byte          // read packet buffer whose capacity is set by user, default value is 64KB
	cConnCount   int32           // number of active client_connections in event-loop
	sConnCount   int32           // number of active server_connections in event-loop
	connections  map[int]*conn   // TCP connection map: fd -> conn
	eventHandler EventHandler    // user eventHandler
	nextTicker   time.Time       // next available ticker time
}

func (el *eventloop) addCConn(delta int32) {
	atomic.AddInt32(&el.cConnCount, delta)
}

func (el *eventloop) loadCConn() int32 {
	return atomic.LoadInt32(&el.cConnCount)
}

func (el *eventloop) addSConn(delta int32) {
	atomic.AddInt32(&el.sConnCount, delta)
}

func (el *eventloop) loadSConn() int32 {
	return atomic.LoadInt32(&el.sConnCount)
}

func (el *eventloop) closeAllSockets() {
	// Close loops and all outstanding connections
	for _, c := range el.connections {
		_ = el.closeConn(c, nil, ConnEof)
	}
}

func (el *eventloop) register(itf interface{}) error {
	c := itf.(*conn)
	if err := el.poller.AddRead(c.pollAttachment); err != nil {
		_ = unix.Close(c.fd)
		c.releaseTCP()
		return err
	}
	el.connections[c.fd] = c
	return el.open(c)
}

func (el *eventloop) open(c *conn) error {
	c.opened = true
	GlobalStats.TotalConnections.WithLabelValues().Inc()

	var out []byte
	var action Action

	switch c.connType {
	case ConnClient:
		el.addCConn(1)
		out, action = el.eventHandler.OnCOpened(c)
	case ConnServer:
		el.addSConn(1)
		out, action = el.eventHandler.OnSOpened(c)
	default:
		logging.Errorf("unknown conn fd %d", c.Fd())
		out, action = nil, Close
	}
	if out != nil {
		if err := c.open(out); err != nil {
			return err
		}
	}

	if !c.outboundBuffer.IsEmpty() {
		if err := el.poller.AddWrite(c.pollAttachment); err != nil {
			return err
		}
	}

	return el.handleAction(c, action)
}

func (el *eventloop) read(c *conn) error {
	n, err := unix.Read(c.fd, el.buffer)
	if err != nil || n == 0 {
		if err == unix.EAGAIN {
			return nil
		}
		if n == 0 {
			return el.closeConn(c, os.NewSyscallError("read", unix.ECONNRESET), ConnEof)
		}
		return el.closeConn(c, os.NewSyscallError("read", err), ConnErr)
	}

	c.buffer = el.buffer[:n]

	switch c.connType {
	case ConnClient:
		return el.cread(c)
	case ConnServer:
		return el.sread(c)
	default:
	}

	logging.Errorf("conn here cannot be none, please check conn: %+v", c)
	return el.closeConn(c, errors.New("conn closed"), ConnErr)
}

func (el *eventloop) cread(c *conn) error {
	for {
		r, err := c.cread()
		if err == codec.ErrInvalidResp {
			logging.Warnf("[%dc] client closed because of invalid resp", c.Fd())
			return el.closeConn(c, nil, ConnErr)
		}
		// incomplete message, waiting for next event polling
		if err != nil {
			break
		}

		out, action := el.eventHandler.OnCReact(r, c)
		if out != nil {
			// Encode data and try to write it back to the peer, this attempt is based on a fact:
			// the peer socket waits for the response data after sending request data to the server,
			// which makes the peer socket writable.
			MsgPool.Put(r)
			if _, err = c.write(out); err != nil {
				return err
			}
		}
		switch action {
		case None:
		case Close:
			return el.closeConn(c, nil, ProxyEof)
		case Shutdown:
			return gerrors.ErrEngineShutdown
		}

		// Check the status of connection every loop since it might be closed
		// during writing data back to the peer due to some kind of system error.
		if !c.opened {
			return nil
		}
	}
	_, _ = c.inboundBuffer.Write(c.buffer)
	return nil
}

func (el *eventloop) sread(s *conn) error {
Loop:
	for {
		r, err := s.sread()
		if err != nil {
			switch err {
			case codec.ErrUnKnown, codec.ErrInvalidResp, codec.ErrInvalidInitializing:
				logging.Errorf("[%ds] redis response parse failed, error: %s", s.fd, err)
				continue

			// process the redis moved/ask packet
			case codec.MovedOrAsk:
				addr, slot := r.parseMovedOrAsk()
				el.eventHandler.OnMoved(addr, slot, s, r)
				continue

			// The current message has been processed, continue to process the next message
			case codec.Continue:
				continue

			// Incomplete message, waiting for next event polling
			case gerrors.ErrIncompletePacket:
				fallthrough
			default:
				break Loop
			}
		}

		if r.Type == codec.RspNeedNtAuth || r.Type == codec.RspNeedAuth || r.Type == codec.RspAuthFailed {
			logging.Errorf("[%dm|%df][%dc|%ds] rcproxy shutdown because of invalid auth, redis response: %s", r.MsgId(), r.Id, r.OwnerFd(), s.fd, r.RspBodyString())
			return gerrors.ErrEngineShutdown
		}

		if r.Owner == nil {
			select {
			case EngineGlobal.clusterChan <- r.RspBody:
			default:
				logging.Warnf("[%dm|%df][%dc|%ds] cluster info channel blocked, cannot write", r.MsgId(), r.Id, r.OwnerFd(), s.fd)
			}
			continue
		}

		var c *conn
		c = r.Owner.(*conn)

		if !c.opened {
			logging.Warnf("[%dm|%df][%dc|%ds] client conn already closed", r.MsgId(), r.Id, r.OwnerFd(), s.fd)
			continue
		}

		if c.inMsgQueue.Empty() {
			logging.Errorf("[%dm|%df][%dc|%ds] redis react happen but client inMsgQueue empty", r.MsgId(), r.Id, r.OwnerFd(), s.fd)
			el.closeConn(c, nil, ProxyEof)
			continue
		}

		// the queued messages have been sent to redis in bulk,
		// and the messages are finally assembled and sent to
		// the client when and only when all the messages have been processed

		// Whether all inMsgQueue messages have been processed
		if !c.inMsgQueue.AllDone() {
			continue
		}

		var bs = make([][]byte, c.inMsgQueue.count)
		bs = bs[:0]
		cur := c.inMsgQueue.head

		var curId uint64
		var curFd = c.fd

		for cur != nil {
			curId = cur.Id
			bs = append(bs, cur.RspBody)
			logging.Debugfunc(func() string { return fmt.Sprintf("[%dm][%dc] got res: %s", cur.Id, c.Fd(), cur.RspBodyString()) })
			cur = cur.prev
		}

		if _, err = c.writev(bs); err != nil {
			logging.Warnf("[%dm][%dc] write to client failed, error: %s, body: %s", cur.Id, c.fd, err, cur.RspBodyString())
			continue
		}

		if !c.opened {
			logging.Warnf("[%dm][%dc] write failed because of client closed", curId, curFd)
			continue
		}

		// release Msg
		for {
			msg := c.dequeueInMsg()
			if msg == nil {
				break
			}
			MsgPool.Put(msg)
		}

		// Check the status of connection every loop since it might be closed
		// during writing data back to the peer due to some kind of system error.
		if !s.opened {
			return nil
		}
	}

	_, _ = s.inboundBuffer.Write(s.buffer)
	return nil
}

const iovMax = 1024

func (el *eventloop) write(c *conn) error {
	iov := c.outboundBuffer.Peek(-1)
	var (
		n   int
		err error
	)
	if len(iov) > 1 {
		if len(iov) > iovMax {
			iov = iov[:iovMax]
		}
		n, err = io.Writev(c.fd, iov)
	} else {
		n, err = unix.Write(c.fd, iov[0])
	}
	_, _ = c.outboundBuffer.Discard(n)
	switch err {
	case nil:
	case unix.EAGAIN:
		return nil
	default:
		return el.closeConn(c, os.NewSyscallError("write", err), ConnErr)
	}

	// All data have been drained, it's no need to monitor the writable events,
	// remove the writable event from poller to help the future event-loops.
	if c.outboundBuffer.IsEmpty() {
		_ = el.poller.ModRead(c.pollAttachment)
	}

	return nil
}

func (el *eventloop) closeConn(c *conn, err error, closeType ConnCloseType) (rerr error) {
	if !c.opened {
		return
	}

	// Send residual data in buffer back to the peer before actually closing the connection.
	if !c.outboundBuffer.IsEmpty() {
		for !c.outboundBuffer.IsEmpty() {
			iov := c.outboundBuffer.Peek(0)
			if len(iov) > iovMax {
				iov = iov[:iovMax]
			}
			if n, e := io.Writev(c.fd, iov); e != nil {
				logging.Warnf("closeConn: error occurs when sending data back to peer, %v", e)
				break
			} else {
				_, _ = c.outboundBuffer.Discard(n)
			}
		}
	}

	err0, err1 := el.poller.Delete(c.fd), unix.Close(c.fd)
	if err0 != nil {
		rerr = fmt.Errorf("failed to delete fd=%d from poller in event-loop(%d): %v", c.fd, el.idx, err0)
	}
	if err1 != nil {
		err1 = fmt.Errorf("failed to close fd=%d in event-loop(%d): %v", c.fd, el.idx, os.NewSyscallError("close", err1))
		if rerr != nil {
			rerr = errors.New(rerr.Error() + " & " + err1.Error())
		} else {
			rerr = err1
		}
	}

	delete(el.connections, c.fd)

	switch c.connType {
	case ConnClient:
		el.eventHandler.OnCClosed(c, err)
		el.addCConn(-1)
		switch closeType {
		case ConnEof:
			GlobalStats.ClientConnectionsClientEof.WithLabelValues().Inc()
		case ConnErr:
			GlobalStats.ClientConnectionsClientErr.WithLabelValues().Inc()
		}
	case ConnServer:
		el.eventHandler.OnSClosed(c, err)
		el.addSConn(-1)
		switch closeType {
		case ConnEof:
			GlobalStats.RedisServerEof.WithLabelValues(c.RemoteAddr()).Inc()
		case ConnErr:
			GlobalStats.RedisServerErr.WithLabelValues(c.RemoteAddr()).Inc()
		}
	default:
		logging.Errorf("unknown conn fd %d", c.Fd())
	}

	c.releaseTCP()

	return
}

func (el *eventloop) ticker() {
	now := time.Now()
	for now.Before(el.nextTicker) {
		return
	}
	el.nextTicker = now.Add(time.Second)

	if EngineGlobal.ClusterNodes.serverChanged {
		logging.Infof("[server changed] start load new server, old redis nodes: %+v", EngineGlobal.ProxyAddrs)

		for k, v := range EngineGlobal.ProxyPool {
			if _, ok := EngineGlobal.ClusterNodes.ServerMap.Get(k); !ok {
				v.Close()
				delete(EngineGlobal.ProxyPool, k)
				logging.Infof("[server changed] remove server %s", k)
			}
		}

		for kv := range EngineGlobal.ClusterNodes.ServerMap.Iter() {
			k := kv.Key.(string)
			v := kv.Value.(*ClusterNode)
			isSlave := v.Role == Slave
			if pool, ok := EngineGlobal.ProxyPool[kv.Key.(string)]; ok {
				pool.SetIsSlave(isSlave)
			} else {
				EngineGlobal.ProxyPool[k] = el.engine.newPool(k, isSlave)
				logging.Infof("[server changed] add new server %s, isSlave: %t", k, isSlave)
			}
		}

		EngineGlobal.Slots2Node.Reset()
		for _, rs := range EngineGlobal.ClusterNodes.Replicasets {
			for _, slotRange := range rs.Master.Slots {
				for i := slotRange.Start; i <= slotRange.End; i++ {
					EngineGlobal.Slots2Node.Set(i, rs)
				}
			}
		}

		EngineGlobal.ProxyAddrs = EngineGlobal.ProxyAddrs[:0]
		for k := range EngineGlobal.ProxyPool {
			EngineGlobal.ProxyAddrs = append(EngineGlobal.ProxyAddrs, k)
		}

		EngineGlobal.ClusterNodes.serverChanged = false
		logging.Infof("[server changed] end load new server, cost: %s, new redis nodes: %+v", time.Since(now), EngineGlobal.ProxyAddrs)
	}

	for k, v := range EngineGlobal.ProxyPool {
		GlobalStats.RedisServerActive.WithLabelValues(k).Set(float64(v.ActiveCount()))
	}

	el.eventHandler.OnTicker()
}

// allow the maximum processing time of redis,
// timeout will report an error to the client
func (el *eventloop) msgTimeout() {
	for {
		frag := getFromTimeoutQueue()
		if frag == nil {
			break
		}
		if frag.Done {
			deleteFromTimeoutQueue(frag)
			continue
		}
		if time.Now().Before(frag.Timeout) {
			break
		}

		deleteFromTimeoutQueue(frag)

		c := frag.Owner
		msg := frag.Peer

		for _, v := range msg.Body {
			if v.Done {
				continue
			}
			v.Error = codec.ErrMsgRequestTimeout
			v.Done = true
		}
		msg.Error = codec.ErrMsgRequestTimeout
		if c == nil || !c.IsOpened() {
			logging.Infof("[%dm|%df][%dc] try to send request timeout but client already closed", frag.MsgId(), frag.Id, frag.OwnerFd())
			continue
		}
		c.AsyncWrite(codec.ErrMsgRequestTimeout.Bytes(), nil)
		logging.Infof("[%dm|%df][%dc] request timeout, consider raising config '[proxy]timeout=%d', send res: %s", frag.MsgId(), frag.Id, frag.OwnerFd(), el.engine.opts.RedisRequestTimeout, codec.ErrMsgRequestTimeout.ShortString())
		c.Discard(0)
	}
}

func (el *eventloop) handleAction(c *conn, action Action) error {
	switch action {
	case None:
		return nil
	case Close:
		return el.closeConn(c, nil, ConnEof)
	case Shutdown:
		return gerrors.ErrEngineShutdown
	default:
		return nil
	}
}
