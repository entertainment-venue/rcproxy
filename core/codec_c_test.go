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
	"rcproxy/core/pkg/hashkit"
	"rcproxy/core/pkg/utils"
)

func TestCRespBodyString(t *testing.T) {
	c := new(Msg)
	c.Body = make(map[int32]*Frag)
	c.Body[0] = &Frag{Req: utils.S2B("*2\r\n$4\r\nmget\r\n$6\r\najioef\r\n")}
	c.Body[11] = &Frag{Req: utils.S2B("*2\r\n$3\r\nget\r\n$4\r\nuser\r\n")}
	if c.BodyString() != "[{ 11 => *2 $3 get $4 user }{ 0 => *2 $4 mget $6 ajioef }]" && c.BodyString() != "[{ 0 => *2 $4 mget $6 ajioef }{ 11 => *2 $3 get $4 user }]" {
		t.Failed()
	}
}

type cRespTest struct {
	Input  string
	Keys   []string
	Expect Msg
	Error  error
}

func TestSDecode(t *testing.T) {
	var cases = [...]cRespTest{
		{
			Input: "*2\r\n$3\r\nget\r\n$3\r\nFoo\r\n",
			Keys:  []string{"Foo"},
			Expect: Msg{
				Type: codec.ReqGet,
				Body: map[int32]*Frag{
					10576: {
						Id:  10576,
						Req: utils.S2B("*2\r\n$3\r\nget\r\n$3\r\nFoo\r\n"),
						Rsp: nil,
						Ok:  false,
					}},
			},
		},
	}

	for _, v := range cases {
		c := new(mockedConn)
		c.On("Peek").Return(utils.S2B(v.Input))

		r := new(CRespCodec)
		r.MsgMaxLength = 64
		cResp, err := r.Decode(c)
		assert.Equal(t, nil, err, "assert err, input: %s", v.Input)
		assert.Equal(t, v.Expect.Type, cResp.Type, "assert type, expect [%s], got [%s], input: %s", codec.Transform2Str(v.Expect.Type), codec.Transform2Str(cResp.Type), v.Input)
		assert.Equal(t, len(v.Expect.Body), len(cResp.Body), "assert len, input: %s", v.Input)

		for _, k := range v.Keys {
			slot := hashkit.Hash(k)
			_, ok := v.Expect.Body[slot]
			assert.Equal(t, true, ok, "assert slot, input: %s", v.Input)
			assert.Equal(t, utils.B2S(v.Expect.Body[slot].Req), v.Input, "assert req, input: %s", v.Input)
		}
	}
}

func initGnetService() {
	s := Engine{
		cCodec: CRespCodec{10000},
		sCodec: SRespCodec{10000},
	}
	EngineGlobal = &s
}

func TestSDecodeMget(t *testing.T) {
	initGnetService()
	var cases = [...]cRespTest{
		{
			Input: "*3\r\n$4\r\nmget\r\n$3\r\nFoo\r\n$3\r\nBar\r\n",
			Keys:  []string{"Foo", "Bar"},
			Expect: Msg{
				Type:  codec.ReqMget,
				Keys:  []string{"Foo", "Bar"},
				Frags: map[int32][]string{5379: {"Bar"}, 10576: {"Foo"}},
				Body: map[int32]*Frag{
					5379: {
						Id:  5379,
						Req: utils.S2B("*2\r\n$4\r\nmget\r\n$3\r\nBar\r\n"),
					},
					10576: {
						Id:  10576,
						Req: utils.S2B("*2\r\n$4\r\nmget\r\n$3\r\nFoo\r\n"),
					},
				},
			},
		},
	}

	for _, v := range cases {
		c := new(mockedConn)
		c.On("Peek").Return(utils.S2B(v.Input))

		r := new(CRespCodec)
		r.MsgMaxLength = 64
		cResp, err := r.Decode(c)
		assert.Equal(t, nil, err, "assert err, input: %s", v.Input)
		assert.Equal(t, v.Expect.Type, cResp.Type, "assert type, expect [%s], got [%s], input: %s", codec.Transform2Str(v.Expect.Type), codec.Transform2Str(cResp.Type), v.Input)
		assert.Equal(t, len(v.Expect.Keys), len(cResp.Keys), "assert keys, input: %s", v.Input)
		assert.Equal(t, len(v.Expect.Body), len(cResp.Body), "assert len, input: %s", v.Input)

		for _, k := range v.Keys {
			slot := hashkit.Hash(k)
			_, ok := v.Expect.Body[slot]
			assert.Equal(t, true, ok, "assert slot, input: %s", v.Input)
			assert.Equal(t, v.Expect.Body[slot].Req, cResp.Body[slot].Req, "assert body.req, input: %s", v.Input)
			assert.Equal(t, v.Expect.Frags[slot], cResp.Frags[slot], "assert frags, slot: %d, input: %s", slot, v.Input)
		}
	}
}

func TestSDecodeDel(t *testing.T) {
	var cases = [...]cRespTest{
		{
			Input: "*3\r\n$3\r\ndel\r\n$3\r\nFoo\r\n$3\r\nBar\r\n",
			Keys:  []string{"Foo", "Bar"},
			Expect: Msg{
				Type:    codec.ReqDel,
				Keys:    []string{"Foo", "Bar"},
				Fd2Slot: map[int]int32{10: 5739, 11: 10576},
				Frags:   map[int32][]string{5379: {"Bar"}, 10576: {"Foo"}},
				Body: map[int32]*Frag{
					5379: {
						Id:  5379,
						Req: utils.S2B("*2\r\n$3\r\ndel\r\n$3\r\nBar\r\n"),
					},
					10576: {
						Id:  10576,
						Req: utils.S2B("*2\r\n$3\r\ndel\r\n$3\r\nFoo\r\n"),
					},
				},
			},
		},
	}

	for _, v := range cases {
		c := new(mockedConn)
		c.On("Peek").Return(utils.S2B(v.Input))

		r := new(CRespCodec)
		r.MsgMaxLength = 64
		cResp, err := r.Decode(c)
		assert.Equal(t, nil, err, "assert err, input: %s", v.Input)
		assert.Equal(t, v.Expect.Type, cResp.Type, "assert type, expect [%s], got [%s], input: %s", codec.Transform2Str(v.Expect.Type), codec.Transform2Str(cResp.Type), v.Input)
		assert.Equal(t, len(v.Expect.Keys), len(cResp.Keys), "assert keys, input: %s", v.Input)
		assert.Equal(t, len(v.Expect.Body), len(cResp.Body), "assert len, input: %s", v.Input)

		for _, k := range v.Keys {
			slot := hashkit.Hash(k)
			_, ok := v.Expect.Body[slot]
			assert.Equal(t, true, ok, "assert slot, input: %s", v.Input)
			assert.Equal(t, v.Expect.Body[slot].Req, cResp.Body[slot].Req, "assert body.req, input: %s", v.Input)
			assert.Equal(t, v.Expect.Frags[slot], cResp.Frags[slot], "assert frags, slot: %d, input: %s", slot, v.Input)
		}
	}
}

func TestSDecodeMset(t *testing.T) {
	var cases = [...]cRespTest{
		{
			Input: "*5\r\n$4\r\nmset\r\n$3\r\nFoo\r\n$3\r\nfoo\r\n$3\r\nBar\r\n$3\r\nbar\r\n",
			Keys:  []string{"Foo", "Bar"},
			Expect: Msg{
				Type:   codec.ReqMset,
				Keys:   []string{"Foo", "Bar"},
				Frags2: map[int32][][2]string{5379: {{"Bar", "bar"}}, 10576: {{"Foo", "foo"}}},
				Body: map[int32]*Frag{
					5379: {
						Id:  5379,
						Req: utils.S2B("*3\r\n$4\r\nmset\r\n$3\r\nBar\r\n$3\r\nbar\r\n"),
					},
					10576: {
						Id:  10576,
						Req: utils.S2B("*3\r\n$4\r\nmset\r\n$3\r\nFoo\r\n$3\r\nfoo\r\n"),
					},
				},
			},
		},
	}

	for _, v := range cases {
		c := new(mockedConn)
		c.On("Peek").Return(utils.S2B(v.Input))

		r := new(CRespCodec)
		r.MsgMaxLength = 64
		cResp, err := r.Decode(c)
		assert.Equal(t, nil, err, "assert err, input: %s", v.Input)
		assert.Equal(t, v.Expect.Type, cResp.Type, "assert type, expect [%s], got [%s], input: %s", codec.Transform2Str(v.Expect.Type), codec.Transform2Str(cResp.Type), v.Input)
		assert.Equal(t, len(v.Expect.Keys), len(cResp.Keys), "assert keys, input: %s", v.Input)
		assert.Equal(t, len(v.Expect.Body), len(cResp.Body), "assert body len, input: %s", v.Input)

		for _, k := range v.Keys {
			slot := hashkit.Hash(k)
			_, ok := v.Expect.Body[slot]
			assert.Equal(t, true, ok, "assert slot, input: %s", v.Input)
			assert.Equal(t, v.Expect.Body[slot].Req, cResp.Body[slot].Req, "assert body.req, input: %s", v.Input)
			assert.Equal(t, v.Expect.Frags[slot], cResp.Frags[slot], "assert frags, slot: %d, input: %s", slot, v.Input)
		}
	}
}