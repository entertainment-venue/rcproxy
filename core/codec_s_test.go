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

package core

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"rcproxy/core/codec"
	"rcproxy/core/pkg/utils"
)

func TestParseMoved(t *testing.T) {
	f := new(Frag)
	f.Type = codec.RspMoved
	f.RspBody = utils.S2B("-MOVED 15495 127.0.0.1:8000\r\n")
	addr, slot := f.parseMovedOrAsk()
	assert.Equal(t, "127.0.0.1:8000", addr)
	assert.Equal(t, int32(15495), slot)
}

func TestParseAsk(t *testing.T) {
	f := new(Frag)
	f.Type = codec.RspAsk
	f.RspBody = utils.S2B("-ASK 15495 127.0.0.1:8000\r\n")
	addr, slot := f.parseMovedOrAsk()
	assert.Equal(t, "127.0.0.1:8000", addr)
	assert.Equal(t, int32(15495), slot)
}

type sRespTest struct {
	Fd     int
	Input  string
	Expect Msg
	Error  error
}

func TestSDecodeSuccess(t *testing.T) {
	var cases = [...]sRespTest{
		{Input: "+OK\r\n", Expect: Msg{Type: codec.RspOk}},
		{Input: "+PONG\r\n", Expect: Msg{Type: codec.RspPong}},

		{Input: "-NOAUTH Authentication required\r\n", Expect: Msg{Type: codec.RspNeedAuth}},
		{Input: "-ERR invalid password\r\n", Expect: Msg{Type: codec.RspAuthFailed}},
		{Input: "-ERR Client sent AUTH, but no password is set\r\n", Expect: Msg{Type: codec.RspNeedNtAuth}},
		{Input: "-MOVED\r\n", Expect: Msg{Type: codec.RspMoved}},
		{Input: "-ASK\r\n", Expect: Msg{Type: codec.RspAsk}},

		{Input: "$1\r\n1\r\n", Expect: Msg{Type: codec.RspBulk}},
		{Input: "$1\r\n1\r\n$2", Expect: Msg{Type: codec.RspBulk, RspBody: utils.S2B("$1\r\n1\r\n")}},

		{Input: "*0\r\n", Expect: Msg{Type: codec.RspMultibulk}},
		{Input: "*1\r\n$3\r\nfoo\r\najfioejfoejaeojf", Expect: Msg{Type: codec.RspMultibulk, RspBody: utils.S2B("*1\r\n$3\r\nfoo\r\n")}},
	}

	for _, v := range cases {
		c := new(mockedConn)
		c.On("Peek").Return(utils.S2B(v.Input))
		c.On("Fd").Return(10)
		c.On("DequeueInFrag").Return(&Frag{})

		r := new(SRespCodec)
		r.MsgMaxLength = 64
		sResp, err := r.Decode(c)

		assert.Equal(t, nil, err, "assert err, input: %s", v.Input)
		assert.Equal(t, v.Expect.Type, sResp.Type, "assert type, expect [%s], got [%s], input: %s", codec.Transform2Str(v.Expect.Type), codec.Transform2Str(sResp.Type), v.Input)
		assert.Equal(t, v.Expect.Error, sResp.Error, "assert error, expect [%s], got [%s], input: %s", v.Expect.Error, sResp.Error, v.Input)
		if len(v.Expect.RspBody) > 0 {
			assert.Equal(t, v.Expect.RspBody, sResp.RspBody, "assert body, input: %s", v.Input)
		} else {
			assert.Equal(t, v.Input, utils.B2S(sResp.RspBody), "assert body, input: %s", v.Input)
		}
	}
}

func TestCDecodeError(t *testing.T) {
	var cases = [...]sRespTest{
		{Input: "+OK", Error: codec.ErrLFNotFound},
		{Input: "+OK\r", Error: codec.ErrLFNotFound},
		{Input: "+OK\n", Error: codec.ErrCRNotFound},
		{Input: "$1\r\n", Error: codec.EmptyLine},
		{Input: "$1\r\na", Error: codec.EmptyLine},
		{Input: "*1\r\n", Error: codec.EmptyLine},
		{Input: "*1\r\n$2\r\na", Error: codec.ShortLine},
	}

	for _, v := range cases {
		c := new(mockedConn)
		c.On("Peek").Return(utils.S2B(v.Input))
		c.On("Fd").Return(10)

		r := new(SRespCodec)
		r.MsgMaxLength = 102400
		_, err := r.Decode(c)
		assert.Equal(t, v.Error, err, "assert error failed, input: %s", v.Input)
	}
}