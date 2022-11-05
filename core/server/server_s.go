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
	"rcproxy/core"
	"rcproxy/core/pkg/logging"
	"rcproxy/core/pkg/utils"
)

// When opening a redis slave connection, must send the READONLY directive before you can access
const ReadOnly = "*1\r\n$8\r\nREADONLY\r\n"

// OnSOpened fires when a new redis server connection has been opened.
func (ls *listenServer) OnSOpened(s core.SConn) (out []byte, action core.Action) {
	logging.Debugf("[%ds] conn open, local: %s, remote: %s", s.Fd(), s.LocalAddr(), s.RemoteAddr())

	var initCmd string
	var step int8

	if len(authCmd) > 0 {
		step++
		initCmd += authCmd
	}

	if s.IsSlave() {
		step++
		initCmd += ReadOnly
	}

	if len(initCmd) > 0 {
		logging.Debugf("[%ds] initializing", s.Fd())
		s.SetInitializeStep(step)
		s.SetInitializeStatus(core.Initializing)
		return utils.S2B(initCmd), core.None
	}

	s.SetInitializeStep(0)
	s.SetInitializeStatus(core.Initialized)
	return nil, core.None
}

// OnSClosed fires when a redis server connection has been closed.
func (ls *listenServer) OnSClosed(s core.SConn, err error) {
	for {
		frag := s.DequeueInFrag()
		if frag == nil {
			break
		}
		if frag.Owner == nil || !frag.Owner.IsOpened() {
			continue
		}
		if frag.Done || frag.Peer.Done {
			continue
		}
		logging.Errorf("[%dm|%df][%dc|%ds] redis server closed, record the client conn", frag.MsgId(), frag.Id, frag.OwnerFd(), s.Fd())
	}
	logging.Infof("[%ds] server conn closed, local: %s, remote: %s, error: %+v", s.Fd(), s.LocalAddr(), s.RemoteAddr(), err)
}
