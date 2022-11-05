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
	"errors"
	"time"

	"rcproxy/core/pkg/logging"
)

type Pool struct {
	Dial func(addr string, isSlave bool) (SConn, error)

	Addr string

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
}

func (eng *engine) newPool(addr string, isSlave bool) *Pool {
	return &Pool{
		Addr:         addr,
		Dial:         eng.Dial,
		isSlave:      isSlave,
		maxActive:    eng.opts.RedisServerConnections,
		AutoBanFlag:  false,
		LiftBanOrder: 0,
	}
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
