// Copyright (c) 2022 The rcproxy Authors
// Copyright (c) 2019 Andy Pan
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

//go:build !poll_opt
// +build !poll_opt

package core

import (
	"runtime"

	"rcproxy/core/internal/netpoll"
	"rcproxy/core/pkg/logging"
)

func (el *eventloop) callback(fd int, ev uint32) error {
	if c, ok := el.connections[fd]; ok {
		// Don't change the ordering of processing EPOLLOUT | EPOLLRDHUP / EPOLLIN unless you're 100%
		// sure what you're doing!
		// Re-ordering can easily introduce bugs and bad side-effects, as I found out painfully in the past.

		// We should always check for the EPOLLOUT event first, as we must try to send the leftover data back to
		// the peer when any error occurs on a connection.
		//
		// Either an EPOLLOUT or EPOLLERR event may be fired when a connection is refused.
		// In either case write() should take care of it properly:
		// 1) writing data back,
		// 2) closing the connection.
		if ev&netpoll.OutEvents != 0 && !c.outboundBuffer.IsEmpty() {
			if err := el.write(c); err != nil {
				return err
			}
		}
		if ev&netpoll.InEvents != 0 {
			return el.read(c)
		}
		return nil
	}
	return el.accept(fd, ev)
}

func (el *eventloop) run() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	defer func() {
		el.closeAllSockets()
		el.ln.close()
		el.engine.signalShutdown()
	}()

	err := el.poller.Polling(el.callback, el.ticker, el.msgTimeout)
	logging.Debugf("event-loop(%d) is exiting due to error: %v", el.idx, err)
}
