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

package codec

import (
	"errors"

	"rcproxy/core/pkg/utils"
)

var ErrCRNotFound = errors.New("there is no \\r")
var ErrLFNotFound = errors.New("there is no \\n")
var BadLine = errors.New("bad response line")
var ShortLine = errors.New("short line")
var EmptyLine = errors.New("empty line")
var Continue = errors.New("continue")
var MovedOrAsk = errors.New("moved or ask")
var AddrNotFound = errors.New("unknown addr")
var UnKnownProxyPool = errors.New("unknown pool")
var UnKnownProxyPoolConn = errors.New("unknown pool conn")
var ErrInvalidResp = errors.New("invalid resp")
var ErrInvalidInitializing = errors.New("invalid initializing")

const (
	OK   Status = "+OK\r\n"
	PONG Status = "+PONG\r\n"
)

const (
	ErrUnKnown                    Error = "-ERR unknown error\r\n"
	ErrAddrNotFoundError          Error = "-ERR addr not found\r\n"
	ErrUnKnownCommand             Error = "-ERR unknown command\r\n"
	ErrUnKnownSlot                Error = "-ERR unknown slot\r\n"
	ErrUnKnownProxyPoolError      Error = "-ERR unknown proxy pool\r\n"
	ErrUnKnownProxyPoolConnError  Error = "-ERR unknown proxy pool conn\r\n"
	ErrUnKnownMget                Error = "-ERR unknown mget error\r\n"
	ErrMsgReqTooLarge             Error = "-ERR req msg length too large\r\n"
	ErrMsgRspTooLarge             Error = "-ERR rsp msg length too large\r\n"
	ErrMsgReqWrongArgumentsNumber Error = "-ERR wrong number of arguments\r\n"
	ErrMsgRequestTimeout          Error = "-ERR proxy request timeout\r\n"
	ErrAuthInvalidPassword        Error = "-ERR invalid password\r\n"
	ErrAuthNeedNtPassword         Error = "-ERR Client sent AUTH, but no password is set\r\n"
)

type Error string

func (err Error) Nil() bool           { return len(err) < 1 }
func (err Error) NotNil() bool        { return len(err) > 0 }
func (err Error) Error() string       { return string(err) }
func (err Error) Bytes() []byte       { return utils.S2B(string(err)) }
func (err Error) String() string      { return string(err) }
func (err Error) ShortString() string { return string(err)[:len(err)-2] }

type Status string

func (s Status) String() string      { return string(s) }
func (s Status) Bytes() []byte       { return utils.S2B(string(s)) }
func (s Status) Len() int            { return len(s) }
func (s Status) ShortString() string { return string(s)[:len(s)-2] }
