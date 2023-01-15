// Copyright (c) 2022 The rcproxy Authors
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

package server

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"rcproxy/core"
	"rcproxy/core/authip"
	"rcproxy/core/codec"
	"rcproxy/core/pkg/logging"
)

// OnCOpened fires when a new client connection has been opened.
func (ls *listenServer) OnCOpened(c core.CConn) (out []byte, action core.Action) {
	access := strings.Split(c.RemoteAddr(), ":")
	if !authip.IpMap.Validate(access[0]) {
		logging.Warnf("[%dc] unauthorized access from %s", c.Fd(), access[0])
		return nil, core.Close
	}

	logging.Debugf("[%dc] conn open, local: %s, remote: %s", c.Fd(), c.LocalAddr(), c.RemoteAddr())
	return nil, core.None
}

// OnCReact fires when a client socket receives data from the peer.
func (ls *listenServer) OnCReact(r *core.Msg, c core.CConn) (out []byte, action core.Action) {
	logging.Debugfunc(func() string { return fmt.Sprintf("[%dm][%dc] got req: %s", r.Id, c.Fd(), r.BodyString()) })

	if r.Type <= codec.UNKNOWN || r.Type >= codec.Sentinel {
		logging.Warnf("[%dm][%dc] unknown command, type: %d, body: %s", r.Id, c.Fd(), r.Type, r.BodyString())
		return codec.ErrUnKnownCommand.Bytes(), core.None
	}

	switch r.Type {
	case codec.ReqTooLarge:
		logging.Infof("[%dm][%dc] request message too large", r.Id, c.Fd())
		return codec.ErrMsgReqTooLarge.Bytes(), core.None
	case codec.ReqWrongArgumentsNumber:
		logging.Infof("[%dm][%dc] wrong arguments number, type: %d, body: %s", r.Id, c.Fd(), r.Type, r.BodyString())
		return codec.ErrMsgReqWrongArgumentsNumber.Bytes(), core.None
	case codec.ReqPing:
		logging.Debugf("[%dm][%dc] got res: [ +PONG ]", r.Id, c.Fd())
		return codec.PONG.Bytes(), core.None
	case codec.ReqQuit:
		logging.Debugf("[%dm][%dc] got res: [ +OK ]", r.Id, c.Fd())
		return codec.OK.Bytes(), core.Close
	}

	core.GlobalStats.ReqCmdIncr(r.Type)

	for slot, frag := range r.Body {
		if r.Type == codec.ReqAuth {
			if len(ls.Password) < 1 {
				return codec.ErrAuthNeedNtPassword.Bytes(), core.None
			}
			if ls.Password != frag.Key {
				return codec.ErrAuthInvalidPassword.Bytes(), core.None
			}
			return codec.OK.Bytes(), core.None
		}
		if core.EngineGlobal.Slots2Node.NotExist(slot) {
			logging.Errorf("[%dm|%df][%dc] waiting for slot loading, type: %d, body: %s", r.Id, frag.Id, c.Fd(), r.Type, frag.ReqString())
			return codec.ErrUnKnownSlot.Bytes(), core.None
		}
		sConn, err, retry, addr := ls.getConn(r, slot)
		if err != nil {
			if retry {
				sConn, err, _, addr = ls.getConn(r, slot)
			}
			if err != nil {
				switch err {
				case codec.AddrNotFound:
					logging.Errorf("[%dm|%df][%dc] unknown redis server, type: %d, body: %s", r.Id, frag.Id, c.Fd(), r.Type, frag.Req)
					return codec.ErrAddrNotFoundError.Bytes(), core.None
				case codec.UnKnownProxyPool:
					logging.Errorf("[%dm|%df][%dc] unknown redis node %s", r.Id, frag.Id, c.Fd(), addr)
					return codec.ErrUnKnownProxyPoolError.Bytes(), core.None
				case codec.UnKnownProxyPoolConn:
					logging.Errorf("[%dm|%df][%dc] redis node %s dial failed", r.Id, frag.Id, c.Fd(), addr)
					return codec.ErrUnKnownProxyPoolConnError.Bytes(), core.None
				}
				logging.Errorf("[%dm|%df][%dc] unknown getConn %s error, please check here, err: %s", r.Id, frag.Id, c.Fd(), addr, err)
				return codec.ErrUnKnown.Bytes(), core.None
			}
		}
		frag.Owner = c

		logging.Debugfunc(func() string {
			return fmt.Sprintf("[%dm|%df][%dc|%ds] key '%s' maps to server '%s' in slot %d", r.Id, frag.Id, c.Fd(), sConn.Fd(), frag.Key, addr, slot)
		})

		sConn.EnqueueOutFrag(frag)
	}

	c.EnqueueInMsg(r)
	return
}

// getConn Get an available connection from the redis connection pool
func (ls *listenServer) getConn(r *core.Msg, slot int32) (core.SConn, error, bool, string) {
	addr, isSlave := ls.route(r, slot)
	if len(addr) < 1 {
		return nil, codec.AddrNotFound, isSlave, ""
	}

	pool, ok := core.EngineGlobal.ProxyPool[addr]
	if !ok {
		return nil, codec.UnKnownProxyPool, isSlave, addr
	}

	conn := pool.Get()
	if conn == nil {
		pool.LiftBanTime = time.Now().Add(time.Duration(ls.ServerRetryTimeout) * time.Duration(1<<pool.LiftBanOrder) * time.Millisecond)
		if pool.LiftBanOrder >= 5 {
			pool.LiftBanOrder = 5
		} else {
			pool.LiftBanOrder++
		}
		pool.AutoBanFlag = true
		logging.Errorf("[%dm] addr %s disconnected, baned for period", r.Id, addr)
		return nil, codec.UnKnownProxyPoolConn, isSlave, addr
	}
	pool.LiftBanOrder = 0
	return conn, nil, false, addr
}

// liveSlaves to avoid frequent memory alloc, set liveSlaves as a global variable
// The main process is a single-threaded service, so don't worry about the concurrency safety
var liveSlaves []string

func (ls *listenServer) route(r *core.Msg, slot int32) (string, bool) {
	if ls.DisableSlave {
		return core.EngineGlobal.Slots2Node.Get(slot).Master.Addr, false
	}
	if r.Type > codec.ReqWriteCmdStart {
		return core.EngineGlobal.Slots2Node.Get(slot).Master.Addr, false
	}

	liveSlaves = liveSlaves[:0]

	for _, v := range core.EngineGlobal.Slots2Node.Get(slot).Slaves {
		pool, ok := core.EngineGlobal.ProxyPool[v.Addr]
		if !ok {
			logging.Warnf("[%dm] redis pool %s not found", r.Id, v.Addr)
			continue
		}

		if pool.AutoBanFlag {
			if pool.LiftBanTime.Before(time.Now()) {
				logging.Warnf("[%dm] addr %s ever disconnected, don't cost ban period, skip this slave!", r.Id, v.Addr)
				continue
			} else {
				logging.Warnf("[%dm] addr %s ever disconnected, cost ban period, pick up it to live slaves!", r.Id, v.Addr)
				pool.AutoBanFlag = false
				liveSlaves = append(liveSlaves, v.Addr)
			}
		} else {
			pool.AutoBanFlag = false
			liveSlaves = append(liveSlaves, v.Addr)
		}

		if len(liveSlaves) == 0 {
			continue
		}

		return liveSlaves[rand.Intn(len(liveSlaves))], true
	}

	return core.EngineGlobal.Slots2Node.Get(slot).Master.Addr, false
}

// OnMoved process the redis moved/ask packet
func (ls *listenServer) OnMoved(addr string, slot int32, s core.SConn, f *core.Frag) {
	f.RspBody = f.RspBody[:0]

	logging.Infof("[%dm|%df][%dc|%ds] moved/ask happen, old_addr: %s new_addr: %s, slot: %d, req: %s",
		f.MsgId(), f.Id, f.OwnerFd(), s.Fd(),
		s.RemoteAddr(), addr, slot, f.ReqString())

	pool, ok := core.EngineGlobal.ProxyPool[addr]
	if !ok {
		logging.Errorf("[%dm|%df][%dc|%ds] moved/ask happen, proxy pool get addr %s failed",
			f.MsgId(), f.Id, f.OwnerFd(), s.Fd(), addr)
		return
	}

	sConn := pool.Get()
	if sConn == nil {
		logging.Errorf("[%dm|%df][%dc|%ds] proxy dial %s failed",
			f.MsgId(), f.Id, f.OwnerFd(), s.Fd(), addr)
		return
	}

	delete(f.Peer.Fd2Slot, s.Fd())
	f.Peer.Fd2Slot[sConn.Fd()] = slot

	sConn.EnqueueOutFrag(f)
}

// OnCClosed fires when a client connection has been closed.
func (ls *listenServer) OnCClosed(c core.CConn, err error) {
	logging.Debugf("[%dc] client conn closed, local: %s, remote: %s", c.Fd(), c.LocalAddr(), c.RemoteAddr())
}
