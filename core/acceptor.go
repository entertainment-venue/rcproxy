// Copyright (c) 2022 The rcproxy Authors
// Copyright (c) 2021 Andy Pan
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
	"os"
	"time"

	"golang.org/x/sys/unix"

	"rcproxy/core/internal/netpoll"
	"rcproxy/core/internal/socket"
	"rcproxy/core/pkg/logging"
)

func (el *eventloop) accept(_ int, _ netpoll.IOEvent) error {
	nfd, sa, err := unix.Accept(el.ln.fd)
	if err != nil {
		if err == unix.EAGAIN {
			return nil
		}
		logging.Errorf("Accept() failed due to error: %v", err)
		return os.NewSyscallError("accept", err)
	}
	if err = os.NewSyscallError("fcntl nonblock", unix.SetNonblock(nfd, true)); err != nil {
		return err
	}

	remoteAddr := socket.SockaddrToTCPOrUnixAddr(sa)
	if el.engine.opts.TCPKeepAlive > 0 {
		err = socket.SetKeepAlivePeriod(nfd, int(el.engine.opts.TCPKeepAlive/time.Second))
		logging.Error(err)
	}

	c := newTCPConn(nfd, el, el.ln.addr, remoteAddr, ConnClient, Initialized, false)
	if err = el.poller.AddRead(c.pollAttachment); err != nil {
		return err
	}
	el.connections[c.fd] = c
	return el.open(c)
}
