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
	"fmt"
	"strconv"
	"strings"

	"rcproxy/core/codec"
	"rcproxy/core/pkg/errors"
	"rcproxy/core/pkg/hashkit"
	"rcproxy/core/pkg/logging"
	"rcproxy/core/pkg/utils"
)

var ShortcutOK = map[int8]string{
	1: codec.OK.String(),
	2: codec.OK.String() + codec.OK.String(),
}

type SRespCodec struct {
	MsgMaxLength int
}

// When a connection to redis is established, there may be two initialization steps
// 1. If a password is configured, send the AUTH command
// 2. If it is a Slave node, then send the READONLY command
//
// For fast response, the above two steps are combined into a single pipeline
// and sent to redis, which also returns the results of both commands at once
func (rc *SRespCodec) InitializingDecode(s SConn) error {
	bs, _ := s.Peek(0)
	buf := codec.NewBuffer(bs)
	if buf.Empty() {
		return errors.ErrIncompletePacket
	}

	totalStep := s.InitializeStep()

	if totalStep < 1 {
		logging.Errorf("[%ds] unknown initialize total step %d", s.Fd(), totalStep)
		return codec.ErrInvalidInitializing
	}

	if _, ok := ShortcutOK[totalStep]; !ok {
		logging.Errorf("[%ds] unknown initialize total step %d", s.Fd(), totalStep)
		return codec.ErrInvalidInitializing
	}

	if (buf.TotalSize() >= int(totalStep)*codec.OK.Len()) && (strings.HasPrefix(utils.B2S(buf.PeekAll()), ShortcutOK[totalStep])) {
		s.Discard(int(totalStep) * codec.OK.Len())
		s.SetInitializeStatus(Initialized)
		logging.Debugf("[%ds] initialized", s.Fd())
		return nil
	}

	if buf.PeekAll()[0] != '-' && buf.PeekAll()[0] != '+' {
		logging.Errorf("[%ds] unknown initialize response: %s", s.Fd(), utils.FormatRedisRESPMessages(buf.PeekAll()))
		return codec.ErrInvalidInitializing
	}

	if strings.HasPrefix(ShortcutOK[totalStep], utils.B2S(buf.PeekAll())) {
		return errors.ErrIncompletePacket
	}

	return nil
}

func (rc *SRespCodec) Decode(s SConn) (*Frag, error) {
	bs, _ := s.Peek(0)
	buf := codec.NewBuffer(bs)
	if buf.Empty() {
		return nil, errors.ErrIncompletePacket
	}

	rType, err := rc.readReply(buf)
	if err != nil {
		return nil, err
	}

	f := s.DequeueInFrag()
	if f == nil {
		logging.Errorf("[%ds] empty inFragQueue, rsp: %s", s.Fd(), utils.FormatRedisRESPMessages(buf.PeekAll()))
		return nil, codec.ErrUnKnown
	}

	f.Type = rType
	f.RspBody = append(f.RspBody[:0], buf.ReadBuf()...)
	logging.Debugfunc(func() string {
		return fmt.Sprintf("[%dm|%df][%dc|%ds] frag dequeue: %s", f.MsgId(), f.Id, f.OwnerFd(), s.Fd(), f.RspBodyString())
	})

	s.Discard(buf.ReadSize())
	return f, nil
}

func (rc *SRespCodec) readReply(buf *codec.Buffer) (codec.Command, error) {
	line, err := buf.ReadLine()
	if err != nil {
		return codec.UNKNOWN, err
	}
	if len(line) == 0 {
		return codec.UNKNOWN, codec.BadLine
	}
	switch line[0] {
	case '+':
		if strings.HasPrefix(utils.B2S(line), codec.OK.ShortString()) {
			return codec.RspOk, nil
		}
		if strings.HasPrefix(utils.B2S(line), codec.PONG.ShortString()) {
			return codec.RspPong, nil
		}
		return codec.RspStatus, nil
	case ':':
		return codec.RspInteger, nil
	case '-':
		switch {
		case strings.HasPrefix(utils.B2S(line), "-NOAUTH Authentication required"):
			return codec.RspNeedAuth, nil
		case strings.HasPrefix(utils.B2S(line), "-ERR invalid password"):
			return codec.RspAuthFailed, nil
		case strings.HasPrefix(utils.B2S(line), "-ERR Client sent AUTH, but no password is set"):
			fallthrough
		case strings.HasPrefix(utils.B2S(line), "-ERR AUTH <password> called without any password configured for the default user."):
			return codec.RspNeedNtAuth, nil
		case strings.HasPrefix(utils.B2S(line), "-MOVED"):
			return codec.RspMoved, nil
		case strings.HasPrefix(utils.B2S(line), "-ASK"):
			return codec.RspAsk, nil
		}
		return codec.RspError, nil
	case '$':
		n, err := parseLen(line[1:])
		if err != nil {
			return codec.UNKNOWN, err
		}
		if n < 0 {
			return codec.RspBulk, nil
		}
		_, err = buf.ReadN(n)
		if err != nil {
			return codec.UNKNOWN, err
		}
		crlf, err := buf.ReadN(2)
		if err != nil {
			return codec.UNKNOWN, err
		}
		if crlf[0] != '\r' || crlf[1] != '\n' {
			return codec.UNKNOWN, codec.ErrInvalidResp
		}
		return codec.RspBulk, nil
	case '*':
		n, err := parseLen(line[1:])
		if n < 0 || err != nil {
			return codec.UNKNOWN, err
		}
		for i := 0; i < n; i++ {
			_, err := rc.readReply(buf)
			if err != nil {
				return codec.UNKNOWN, err
			}
		}
		return codec.RspMultibulk, nil
	}
	return codec.UNKNOWN, codec.ErrInvalidResp
}

func (rc *SRespCodec) MGet(f *Frag, sfd int) error {
	f.Rsp = rc.parseMGet(f)
	f.Done = true

	if f.Peer.FragDoneNumber < len(f.Peer.Body) {
		logging.Debugf("[%dm|%df][%dc|%ds] mget frag done %d, waiting for other frags", f.MsgId(), f.Id, f.OwnerFd(), sfd, f.Peer.FragDoneNumber)
		return codec.Continue
	}
	logging.Debugf("[%dm|%df][%dc|%ds] all mget frag done %d, prepare to reply client", f.MsgId(), f.Id, f.OwnerFd(), sfd, f.Peer.FragDoneNumber)

	msg := f.Peer
	msg.Done = true
	msg.RspBody = append(msg.RspBody[:0], '*')
	msg.RspBody = append(msg.RspBody, strconv.Itoa(len(msg.Keys))...)
	msg.RspBody = append(msg.RspBody, codec.LFCRByte...)

	for _, k := range msg.Keys {
		slot := hashkit.Hash(k)
		for i, v := range msg.Frags[slot] {
			if v == k {
				msg.RspBody = append(msg.RspBody, msg.Body[slot].Rsp[i]...)
				break
			}
		}
	}

	if len(msg.RspBody) > rc.MsgMaxLength {
		msg.Error = codec.ErrMsgRspTooLarge
		msg.RspBody = append(msg.RspBody[:0], codec.ErrMsgRspTooLarge.Bytes()...)
		return nil
	}
	return nil
}

func (rc *SRespCodec) MSet(f *Frag, sfd int) error {
	f.Ok = f.Type == codec.RspOk
	f.Done = true

	if !f.Ok {
		logging.Warnf("unknown mset error, msg: %+v", f)
	}

	if f.Peer.FragDoneNumber < len(f.Peer.Body) {
		logging.Debugf("[%dm|%df][%dc|%ds] mset frag done %d, waiting for other frags", f.MsgId(), f.Id, f.OwnerFd(), sfd, f.Peer.FragDoneNumber)
		return codec.Continue
	}
	logging.Debugf("[%dm|%df][%dc|%ds] all mset frag done %d, prepare to reply client", f.MsgId(), f.Id, f.OwnerFd(), sfd, f.Peer.FragDoneNumber)

	msg := f.Peer
	msg.Done = true
	for _, v := range msg.Body {
		if !v.Ok {
			logging.Warnf("unknown mset error, msg: %+v", msg)
			msg.RspBody = append(msg.RspBody[:0], codec.ErrUnKnown.Bytes()...)
			return nil
		}
	}
	msg.RspBody = append(msg.RspBody[:0], codec.OK...)
	return nil
}

func (rc *SRespCodec) Del(f *Frag, sfd int) error {
	line := f.RspBody[1 : len(f.RspBody)-2]
	n, _ := parseLen(line)
	f.Peer.DelNum += n
	f.Done = true

	if f.Peer.FragDoneNumber < len(f.Peer.Body) {
		logging.Debugf("[%dm|%df][%dc|%ds] del frag done %d, waiting for other frags", f.MsgId(), f.Id, f.OwnerFd(), sfd, f.Peer.FragDoneNumber)
		return codec.Continue
	}
	logging.Debugf("[%dm|%df][%dc|%ds] all del frag done %d, prepare to reply client", f.MsgId(), f.Id, f.OwnerFd(), sfd, f.Peer.FragDoneNumber)

	msg := f.Peer
	msg.Done = true
	msg.RspBody = append(msg.RspBody[:0], fmt.Sprintf(":%d\r\n", msg.DelNum)...)
	return nil
}

func (rc *SRespCodec) Default(f *Frag) error {
	f.Done = true
	msg := f.Peer
	msg.Done = true
	if len(f.RspBody) > rc.MsgMaxLength {
		msg.Error = codec.ErrMsgRspTooLarge
		msg.RspBody = append(msg.RspBody[:0], codec.ErrMsgRspTooLarge.Bytes()...)
		return nil
	}
	msg.RspBody = append(msg.RspBody[:0], f.RspBody...)
	return nil
}

func (rc *SRespCodec) parseMGet(f *Frag) []string {
	buf := codec.NewBuffer(f.RspBody)

	kLenBytes, _ := buf.ReadLine()
	kLen, _ := parseLen(kLenBytes[1:])
	var msg = make([]string, kLen)
	msg = msg[:0]

	for {
		line, err := buf.ReadLine()
		if err != nil {
			return msg
		}
		n, _ := parseLen(line[1:])
		if n < 0 {
			msg = append(msg, fmt.Sprintf("%s\r\n", line))
			continue
		}
		v, _ := buf.ReadLine()
		msg = append(msg, fmt.Sprintf("%s\r\n%s\r\n", line, v))
	}
}

func (rc *SRespCodec) sizeTooLarge(size int) bool {
	if size > rc.MsgMaxLength {
		return true
	}
	return false
}
