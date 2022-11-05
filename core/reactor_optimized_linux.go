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

//go:build poll_opt
// +build poll_opt

package core

import (
	"runtime"

	"rcproxy/core/pkg/logging"
)

func (el *eventloop) run() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	defer func() {
		el.closeAllSockets()
		el.ln.close()
		el.engine.signalShutdown()
	}()

	err := el.poller.Polling(el.ticker, el.msgTimeout)
	logging.Debugf("event-loop(%d) is exiting due to error: %v", el.idx, err)
}
