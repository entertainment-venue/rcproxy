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
	"strconv"

	"rcproxy/core"
)

var authCmd string

const AuthCmd = "*2\r\n$4\r\nauth\r\n$%s\r\n%s\r\n"

func NewListenServer(opts ...Option) *listenServer {
	options := loadOptions(opts...)

	server := &listenServer{
		Options: options,
	}
	return server
}

type listenServer struct {
	*core.BuiltinEventEngine

	*Options
}

// OnBoot fires when rcproxy is ready for accepting connections.
func (ls *listenServer) OnBoot(_ core.Engine) (action core.Action) {
	if len(ls.Password) > 0 {
		var passwdLen = strconv.Itoa(len(ls.Password))
		authCmd = fmt.Sprintf(AuthCmd, passwdLen, ls.Password)
	}
	return
}
