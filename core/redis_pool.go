// Copyright (c) 2022 The rcproxy Authors
// Copyright (c) 2011 Twitter, Inc.
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

package core

import (
	"context"
	"errors"
	"time"

	"rcproxy/core/pkg/logging"
	"rcproxy/core/pkg/redis"
)

type Pool struct {
	Dial func(addr string, isSlave bool) (SConn, error)

	Addr   string
	Passwd string

	maxActive int        // maximum number of connections to each redis node.
	active    activeList // active connections. Note that all connections are active.

	// LiftBanOrder if the redis node is continuously offline, add gradient to LiftBanTime here.
	// For example, the initial probe failure is disabled for 1 second,
	// the second probe is disabled for 2 seconds,
	// and the third probe is disabled for 4 seconds.
	// The maximum value of LiftBanOrder is 5.
	LiftBanOrder int32
	LiftBanTime  time.Time // If the redis node is offline, set the remaining disable time.
	AutoBanFlag  bool      // set to true if the redis node is offline.

	isSlave bool // whether it is a slave node.
	closed  bool // set to true when the pool is closed.

	ctx    context.Context
	cancel context.CancelFunc
}

func (eng *engine) newPool(addr string, isSlave bool) *Pool {
	ctx, cancelFunc := context.WithCancel(context.Background())
	p := &Pool{
		Addr:         addr,
		Passwd:       eng.opts.RedisPasswd,
		Dial:         eng.Dial,
		isSlave:      isSlave,
		maxActive:    eng.opts.RedisServerConnections,
		AutoBanFlag:  false,
		LiftBanOrder: 0,
		ctx:          ctx,
		cancel:       cancelFunc,
	}
	go p.monitor()
	return p
}

func (p *Pool) Get() SConn {
	if p.closed {
		logging.Errorf("get on closed pool, addr: %s", p.Addr)
		return nil
	}

	var c SConn
	var err error
	if p.active.count < p.maxActive {
		c, err = p.dial()
		if err != nil {
			logging.Errorf("failed to dial, addr: %s, err: %s", p.Addr, err)
			return nil
		}
		p.active.pushFront(&poolConn{c: c})
		return c
	}

	for {
		if p.active.count < 1 {
			break
		}
		pc := p.active.back
		p.active.popBack()
		if !pc.c.IsOpened() {
			continue
		}
		p.active.pushFront(pc)
		return pc.c
	}

	c, err = p.dial()
	if err != nil {
		logging.Errorf("failed to dial, addr: %s, err: %s", p.Addr, err)
		return nil
	}
	p.active.pushFront(&poolConn{c: c})
	return c
}

// ActiveCount returns the number of active connections in the pool.
// Note that all connections are active
func (p *Pool) ActiveCount() int {
	return p.active.count
}

// Close releases the resources used by the pool.
func (p *Pool) Close() {
	if p.closed {
		return
	}
	p.Release()
	p.closed = true
	p.cancel()
}

// Close releases the resources used by the pool.
func (p *Pool) Release() {
	if p.closed {
		return
	}
	pc := p.active.front
	p.active.count = 0
	p.active.front, p.active.back = nil, nil
	for ; pc != nil; pc = pc.next {
		pc.c.Close()
	}
	return
}

func (p *Pool) dial() (SConn, error) {
	if p.Dial != nil {
		return p.Dial(p.Addr, p.isSlave)
	}
	return nil, errors.New("redigo: must pass Dial or DialContext to pool")
}

func (p *Pool) SetIsSlave(isSlave bool) {
	if p.isSlave != isSlave {
		p.isSlave = isSlave
		p.Release()
		logging.Infof("update server %s set isSlave %t with all conn closed", p.Addr, isSlave)
	}
}

func (p *Pool) monitor() {
	ticker := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			if p.closed {
				return
			}
			err := p.detect()
			if err == nil {
				p.LiftBanOrder = 0
				if p.AutoBanFlag {
					logging.Errorf("[monitor] addr %s reconnected", p.Addr)
				}
				p.AutoBanFlag = false
				break
			} else {
				time.Sleep(5 * time.Second)
				err = p.detect()
				if err == nil {
					p.LiftBanOrder = 0
					if p.AutoBanFlag {
						logging.Errorf("[monitor] addr %s reconnected", p.Addr)
					}
					p.AutoBanFlag = false
					break
				}
			}

			p.LiftBanTime = time.Now().Add(60 * time.Second)
			p.AutoBanFlag = true
			logging.Errorf("[monitor] addr %s disconnected, baned for period, err: %s", p.Addr, err)
		}
	}
}

func (p *Pool) detect() error {
	c, err := redis.Dial(
		p.Addr,
		p.Passwd,
		redis.DialConnectTimeout(1*time.Second),
		redis.DialReadTimeout(3*time.Second),
		redis.DialWriteTimeout(3*time.Second),
	)
	if err != nil {
		return err
	}
	defer c.Close()
	res, err := c.Do("PING")
	if err != nil {
		return err
	}

	if v, ok := res.(string); !ok {
		return errors.New("unknown res")
	} else if v != "PONG" {
		return errors.New("invalid res" + v)
	} else {
		return nil
	}
}

type activeList struct {
	front, back *poolConn
	count       int
}

type poolConn struct {
	c          SConn
	next, prev *poolConn
}

// front -> x -> x -> back
func (l *activeList) pushFront(pc *poolConn) {
	pc.next = l.front
	pc.prev = nil
	if l.count == 0 {
		l.back = pc
	} else {
		l.front.prev = pc
	}
	l.front = pc
	l.count++
}

func (l *activeList) popBack() {
	pc := l.back
	l.count--
	if l.count == 0 {
		l.front, l.back = nil, nil
	} else {
		pc.prev.next = nil
		l.back = pc.prev
	}
	pc.next, pc.prev = nil, nil
}
