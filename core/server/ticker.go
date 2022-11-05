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
	"math/rand"

	"rcproxy/core"
	"rcproxy/core/pkg/logging"
)

// Pick a random redis node every second to send the cluster nodes command
func (ls *listenServer) OnTicker() {
	nAddr := len(core.EngineGlobal.ProxyAddrs)
	if nAddr < 1 {
		logging.Errorf("no addr found")
		return
	}
	addr := core.EngineGlobal.ProxyAddrs[rand.Intn(nAddr)]
	pool, ok := core.EngineGlobal.ProxyPool[addr]
	if !ok {
		logging.Errorf("proxy pool[%s] get failed while ticker", addr)
		return
	}

	sConn := pool.Get()
	if sConn == nil {
		logging.Errorf("proxy.Dial[%s] failed", addr)
		return
	}

	if err := sConn.WriteClusterNodes(); err != nil {
		logging.Errorf("[%ds] failed to write cluster nodes, err: %s", sConn.Fd(), err)
		return
	}
}
