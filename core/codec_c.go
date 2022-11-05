// Copyright (c) 2022 The rcproxy Authors
// Copyright (c) 2011 Twitter, Inc.
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
	"strconv"

	"rcproxy/core/codec"
	"rcproxy/core/pkg/errors"
	"rcproxy/core/pkg/hashkit"
	"rcproxy/core/pkg/logging"
	"rcproxy/core/pkg/utils"
)

type CRespCodec struct {
	MsgMaxLength int
}

// There are three cases of protocol parsing
// 1. successful parsing
// 2. tcp packet incompleteness leads to parsing exceptions, wait for the next event loop
// 3. illegal packets leads to parsing exceptions, so close the client connection directly.
func (rc *CRespCodec) Decode(c CConn) (*Msg, error) {
	bs, _ := c.Peek(0)
	buf := codec.NewBuffer(bs)
	if buf.Empty() {
		return nil, errors.ErrIncompletePacket
	}

	line, err := buf.ReadLine()
	if err != nil {
		return nil, errors.ErrIncompletePacket
	}

	msgId++

	var n int
	switch line[0] {
	case '*':
		n, err = parseLen(line[1:])
		if n < 1 || err != nil {
			logging.Warnf("[%dm][%dc] unexpect resp, buf: %s", msgId, c.Fd(), utils.FormatRedisRESPMessages(buf.PeekAll()))
			return nil, err
		}
	default:
		logging.Warnf("[%dm][%dc] unexpect resp, buf: %s", msgId, c.Fd(), utils.FormatRedisRESPMessages(buf.PeekAll()))
		return nil, codec.ErrInvalidResp
	}

	msg, err := rc.parseLine(buf)
	if err != nil {
		logging.Warnf("[%dm][%dc] unexpect resp, buf: %s", msgId, c.Fd(), utils.FormatRedisRESPMessages(buf.PeekAll()))
		return nil, err
	}
	n--

	resp := MsgPool.Get()
	resp.Id = msgId
	resp.Owner = c
	resp.Type = codec.Transform2Type(msg, n)
	resp.Body = make(map[int32]*Frag, n)
	resp.Fd2Slot = make(map[int]int32, n)

	if rc.sizeTooLarge(buf.TotalSize()) {
		resp.Type = codec.ReqTooLarge
	}

	switch resp.Type {
	case codec.ReqMget:
		if err = rc.Frag1(c, n, resp, buf); err != nil {
			return nil, err
		}
		EngineGlobal.cCodec.MGet(resp)
		GlobalStats.Fragments.WithLabelValues(codec.Transform2Str(codec.ReqMget)).Inc()
	case codec.ReqDel:
		if err = rc.Frag1(c, n, resp, buf); err != nil {
			return nil, err
		}
		EngineGlobal.cCodec.Del(resp)
		GlobalStats.Fragments.WithLabelValues(codec.Transform2Str(codec.ReqDel)).Inc()
	case codec.ReqMset:
		if err = rc.Frag2(c, n, resp, buf); err != nil {
			return nil, err
		}
		EngineGlobal.cCodec.MSet(resp)
		GlobalStats.Fragments.WithLabelValues(codec.Transform2Str(codec.ReqMset)).Inc()
	case codec.ReqEval, codec.ReqEvalsha:
		if err = rc.Eval(c, n, resp, buf); err != nil {
			return nil, err
		}
	default:
		if err = rc.Default(c, n, resp, buf); err != nil {
			return nil, err
		}
	}
	GlobalStats.TotalRequests.WithLabelValues().Inc()
	_, _ = c.Discard(buf.ReadSize())
	return resp, nil
}

func (rc *CRespCodec) Frag1(c CConn, n int, resp *Msg, buf *codec.Buffer) error {
	resp.Frags = make(map[int32][]string, n)
	for i := 0; i < n; i++ {
		msg, err := rc.parseLine(buf)
		if err != nil {
			logging.Warnf("[%dm][%dc] unexpect resp, buf: %s", resp.Id, c.Fd(), utils.FormatRedisRESPMessages(buf.PeekAll()))
			return err
		}
		seg := string(msg)
		resp.Keys = append(resp.Keys, seg)
		slot := hashkit.Hash(seg)
		if v, ok := resp.Frags[slot]; ok {
			resp.Frags[slot] = append(v, seg)
		} else {
			resp.Frags[slot] = []string{seg}
		}
	}
	return nil
}

func (rc *CRespCodec) Frag2(c CConn, n int, resp *Msg, buf *codec.Buffer) error {
	resp.Frags2 = make(map[int32][][2]string, n/2)
	for i := 0; i < n; i = i + 2 {
		msg, err := rc.parseLine(buf)
		if err != nil {
			logging.Warnf("[%dm][%dc] unexpect resp, buf: %s", resp.Id, c.Fd(), utils.FormatRedisRESPMessages(buf.PeekAll()))
			return err
		}
		val, err := rc.parseLine(buf)
		if err != nil {
			logging.Warnf("[%dm][%dc] unexpect resp, buf: %s", resp.Id, c.Fd(), utils.FormatRedisRESPMessages(buf.PeekAll()))
			return err
		}

		seg := string(msg)
		seg2 := string(val)
		segArr := [2]string{seg, seg2}
		resp.Keys = append(resp.Keys, seg)
		slot := hashkit.Hash(seg)
		if v, ok := resp.Frags2[slot]; ok {
			resp.Frags2[slot] = append(v, segArr)
		} else {
			resp.Frags2[slot] = [][2]string{segArr}
		}
	}
	return nil
}

func (rc *CRespCodec) Eval(c CConn, n int, resp *Msg, buf *codec.Buffer) error {
	if n < 3 {
		resp.Type = codec.ReqWrongArgumentsNumber
	}
	var key string
	var slot int32
	for i := 0; i < n; i++ {
		msg, err := rc.parseLine(buf)
		if err != nil {
			logging.Warnf("[%dm][%dc] unexpect resp, buf: %s", resp.Id, c.Fd(), utils.FormatRedisRESPMessages(buf.PeekAll()))
			return err
		}
		if i == 2 {
			key = string(msg)
			slot = hashkit.Hash(key)
		}
	}
	frag := FragPool.Get()
	frag.Key = key
	frag.Peer = resp
	frag.Req = append(frag.Req[:0], buf.ReadBuf()...)
	resp.Body[slot] = frag
	return nil
}

func (rc *CRespCodec) Default(c CConn, n int, resp *Msg, buf *codec.Buffer) error {
	var key string
	var slot int32
	for i := 0; i < n; i++ {
		msg, err := rc.parseLine(buf)
		if err != nil {
			logging.Warnf("[%dm][%dc] unexpect resp, buf: %s", resp.Id, c.Fd(), utils.FormatRedisRESPMessages(buf.PeekAll()))
			return err
		}
		if i == 0 {
			key = string(msg)
			slot = hashkit.Hash(key)
		}
	}
	frag := FragPool.Get()
	frag.Key = key
	frag.Peer = resp
	frag.Req = append(frag.Req[:0], buf.ReadBuf()...)
	resp.Body[slot] = frag
	return nil
}

func (rc *CRespCodec) MGet(resp *Msg) {
	for slot, keys := range resp.Frags {
		frag := FragPool.Get()
		frag.Key = keys[0]
		frag.Peer = resp
		frag.Req = append(frag.Req, '*')
		frag.Req = append(frag.Req, strconv.Itoa(len(keys)+1)...)
		frag.Req = append(frag.Req, "\r\n$4\r\nmget\r\n"...)
		for _, k := range keys {
			frag.Req = append(frag.Req, '$')
			frag.Req = append(frag.Req, strconv.Itoa(len(k))...)
			frag.Req = append(frag.Req, codec.LFCRByte...)
			frag.Req = append(frag.Req, k...)
			frag.Req = append(frag.Req, codec.LFCRByte...)
		}
		resp.Body[slot] = frag
	}
}

func (rc *CRespCodec) Del(resp *Msg) {
	for slot, keys := range resp.Frags {
		frag := FragPool.Get()
		frag.Key = keys[0]
		frag.Peer = resp
		frag.Req = append(frag.Req, '*')
		frag.Req = append(frag.Req, strconv.Itoa(len(keys)+1)...)
		frag.Req = append(frag.Req, "\r\n$3\r\ndel\r\n"...)
		for _, k := range keys {
			frag.Req = append(frag.Req, '$')
			frag.Req = append(frag.Req, strconv.Itoa(len(k))...)
			frag.Req = append(frag.Req, codec.LFCRByte...)
			frag.Req = append(frag.Req, k...)
			frag.Req = append(frag.Req, codec.LFCRByte...)
		}
		resp.Body[slot] = frag
	}
}

func (rc *CRespCodec) MSet(resp *Msg) {
	for slot, keys := range resp.Frags2 {
		frag := FragPool.Get()
		frag.Key = keys[0][0]
		frag.Peer = resp
		frag.Req = append(frag.Req, '*')
		frag.Req = append(frag.Req, strconv.Itoa(len(keys)*2+1)...)
		frag.Req = append(frag.Req, "\r\n$4\r\nmset\r\n"...)
		for _, ks := range keys {
			for _, k := range ks {
				frag.Req = append(frag.Req, '$')
				frag.Req = append(frag.Req, strconv.Itoa(len(k))...)
				frag.Req = append(frag.Req, codec.LFCRByte...)
				frag.Req = append(frag.Req, k...)
				frag.Req = append(frag.Req, codec.LFCRByte...)
			}
		}
		resp.Body[slot] = frag
	}
}

func (rc *CRespCodec) parseLine(buf *codec.Buffer) ([]byte, error) {
	line, err := buf.ReadLine()
	if err != nil {
		return nil, err
	}
	switch line[0] {
	case '$':
		n, err := parseLen(line[1:])
		if n < 0 || err != nil {
			return nil, err
		}
		b, err := buf.ReadN(n)
		if err != nil {
			return nil, err
		}
		crlf, err := buf.ReadN(2)
		if err != nil {
			return nil, codec.ShortLine
		}

		if crlf[0] != '\r' || crlf[1] != '\n' {
			return nil, codec.BadLine
		}
		return b, nil
	default:
		return nil, codec.ErrInvalidResp
	}
}

func (rc *CRespCodec) sizeTooLarge(size int) bool {
	if size > rc.MsgMaxLength {
		return true
	}
	return false
}
