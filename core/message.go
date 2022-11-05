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
	"bytes"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/petar/GoLLRB/llrb"

	"rcproxy/core/codec"
	"rcproxy/core/pkg/constant"
	"rcproxy/core/pkg/logging"
)

// msgId unique identification of the message
var msgId uint64

// fragId unique identification of the frag
var fragId uint64

var timeoutTree *llrb.LLRB
var MsgPool = msgPool{sync.Pool{New: func() interface{} { return new(Msg) }}}
var FragPool = fragPool{}

func init() {
	timeoutTree = llrb.New()
}

type Msg struct {
	prev *Msg
	next *Msg

	Owner CConn // client conn

	Id uint64
	// for request
	Body map[int32]*Frag
	// for response
	RspBody []byte
	Error   codec.Error

	// for frag ref
	Fd2Slot        map[int]int32
	Keys           []string
	Frags          map[int32][]string    // for mget/del
	Frags2         map[int32][][2]string // for mset
	FragDoneNumber int                   // number of finished frags
	DelNum         int                   // for del

	Type codec.Command // request command type
	Done bool          // all frags Done
}

type msgPool struct {
	sync.Pool
}

func (p *msgPool) Get() *Msg {
	msg := p.Pool.Get().(*Msg)
	return msg
}

func (p *msgPool) Put(m *Msg) {
	if m == nil {
		return
	}
	m.Id = 0
	m.Type = codec.UNKNOWN
	m.Owner = nil

	m.Body = nil
	m.RspBody = m.RspBody[:0]
	m.Done = false
	m.Error = ""
	m.Fd2Slot = nil
	m.Keys = m.Keys[:0]
	m.Frags = nil
	m.Frags2 = nil
	m.FragDoneNumber = 0
	m.DelNum = 0

	m.prev = nil
	m.next = nil

	p.Pool.Put(m)
}

// Frag client requests may be split into multiple frag and requested to different redis nodes
type Frag struct {
	prev *Frag
	next *Frag

	Owner CConn
	Peer  *Msg

	Id      uint64
	Time    time.Time
	Timeout time.Time
	Key     string
	Error   codec.Error
	Req     []byte
	RspBody []byte
	Rsp     []string // for mget
	Type    codec.Command
	Ok      bool // for mset
	Done    bool // is the current frag completed
}

func (f *Frag) MsgId() uint64 {
	if f.Peer == nil {
		return 0
	}
	return f.Peer.Id
}

func (f *Frag) slowLogCheck(s SConn) {
	if f.Owner == nil || f.Peer == nil {
		return
	}
	if EngineGlobal.eng.opts.RedisSlowlogSlowerThan < 1 {
		return
	}

	costTime := int64(time.Since(f.Time) / time.Millisecond)
	GlobalStats.Request.WithLabelValues().Observe(float64(costTime))

	if costTime < EngineGlobal.eng.opts.RedisSlowlogSlowerThan {
		return
	}

	logging.Warnf(constant.TitleSlowLog+" [%dm|%df][%dc|%ds] remote_addr=%s redis_addr=%s cost_time=%dms request_type=%s request_len=%d response_len=%d key=%s",
		f.MsgId(), f.Id, f.OwnerFd(), s.Fd(), f.Owner.RemoteAddr(), s.RemoteAddr(), costTime, codec.Transform2Str(f.MsgType()), len(f.Req), len(f.RspBody), f.Key)
}

func (f *Frag) OwnerFd() int {
	if f.Owner == nil {
		return -1
	}
	return f.Owner.Fd()
}

func (f *Frag) MsgType() codec.Command {
	if f.Peer == nil {
		return codec.UNKNOWN
	}
	return f.Peer.Type
}

func (f *Frag) ReqString() string {
	if len(f.Req) < 1 {
		return ""
	}
	var buf bytes.Buffer
	buf.WriteByte('[')
	buf.WriteByte(' ')
	for _, b := range f.Req {
		if b == '\r' {
			continue
		}
		if b == '\n' {
			buf.WriteByte(' ')
			continue
		}
		buf.WriteByte(b)
	}
	buf.WriteByte(']')
	return buf.String()
}

func (f *Frag) RspBodyString() string {
	if len(f.RspBody) < 1 {
		return ""
	}
	var buf bytes.Buffer
	buf.WriteByte('[')
	buf.WriteByte(' ')
	for _, b := range f.RspBody {
		if b == '\r' {
			continue
		}
		if b == '\n' {
			buf.WriteByte(' ')
			continue
		}
		buf.WriteByte(b)
	}
	buf.WriteByte(']')
	return buf.String()
}

type fragPool struct{}

func (p *fragPool) Get() *Frag {
	f := new(Frag)
	fragId++
	f.Id = fragId
	f.Req = make([]byte, 64)
	f.Req = f.Req[:0]
	f.Time = time.Now()
	return f
}

func (m *Msg) BodyString() string {
	if len(m.Body) < 1 {
		return ""
	}
	var buf bytes.Buffer
	buf.WriteByte('[')
	for slot, v := range m.Body {
		buf.WriteByte('{')
		buf.WriteByte(' ')
		buf.WriteString(strconv.FormatUint(uint64(slot), 10))
		buf.WriteByte(' ')
		buf.WriteString("=>")
		buf.WriteByte(' ')
		for _, b := range v.Req {
			if b == '\r' {
				continue
			}
			if b == '\n' {
				buf.WriteByte(' ')
				continue
			}
			buf.WriteByte(b)
		}
		buf.WriteByte('}')
	}
	buf.WriteByte(']')
	return buf.String()
}

func (m *Msg) RspBodyString() string {
	if len(m.RspBody) < 1 {
		return ""
	}
	var buf bytes.Buffer
	buf.WriteByte('[')
	buf.WriteByte(' ')
	for _, b := range m.RspBody {
		if b == '\r' {
			continue
		}
		if b == '\n' {
			buf.WriteByte(' ')
			continue
		}
		buf.WriteByte(b)
	}
	buf.WriteByte(']')
	return buf.String()
}

func (f *Frag) parseMovedOrAsk() (addr string, slot int32) {
	if len(f.RspBody) < 10 {
		return "", 0
	}

	var i int
	switch f.Type {
	case codec.RspMoved:
		i = 7
	case codec.RspAsk:
		i = 5
	default:
		return "", 0
	}

	l := strings.Split(string(f.RspBody[i:len(f.RspBody)-2]), " ")
	if len(l) < 2 {
		return "", 0
	}
	ui, _ := strconv.ParseUint(l[0], 0, 64)
	return l[1], int32(ui)
}

func (f *Frag) Less(than llrb.Item) bool {
	return f.Timeout.Before(than.(*Frag).Timeout)
}

func pushToTimeoutQueue(msg *Frag, timeout int) {
	if timeout <= 0 {
		return
	}
	if msg.Owner == nil || msg.Peer == nil {
		return
	}
	msg.Timeout = time.Now().Add(time.Duration(timeout) * time.Millisecond)
	timeoutTree.ReplaceOrInsert(msg)
}

func popFromTimeoutQueue() *Frag {
	min := timeoutTree.DeleteMin()
	if min == nil {
		return nil
	}
	return min.(*Frag)
}

func getFromTimeoutQueue() *Frag {
	min := timeoutTree.Min()
	if min == nil {
		return nil
	}
	return min.(*Frag)
}

func deleteFromTimeoutQueue(f *Frag) {
	timeoutTree.Delete(f)
}

func deleteMinFromTimeoutQueue() {
	timeoutTree.DeleteMin()
}

func lengthOfTimeoutQueue() float64 {
	return float64(timeoutTree.Len())
}

func depthOfTimeoutQueue() (float64, float64) {
	return timeoutTree.HeightStats()
}

// MsgQueue tail -> x -> x -> head
type MsgQueue struct {
	tail, head *Msg
	count      int
}

func (l *MsgQueue) Reset() {
	l.count = 0
	l.tail = nil
	l.head = nil
}

func (l *MsgQueue) AllDone() bool {
	cur := l.head
	for cur != nil {
		if !cur.Done {
			return false
		}
		cur = cur.prev
	}
	return true
}

func (l *MsgQueue) Empty() bool {
	return l.count < 1
}

func (l *MsgQueue) PushTail(m *Msg) {
	m.next = l.tail
	m.prev = nil
	if l.count == 0 {
		l.head = m
	} else {
		l.tail.prev = m
	}
	l.tail = m
	l.count++
}

func (l *MsgQueue) PopHead() {
	if l.count == 0 {
		return
	}
	m := l.head
	l.count--
	if l.count == 0 {
		l.tail, l.head = nil, nil
	} else {
		m.prev.next = nil
		l.head = m.prev
	}
	m.next, m.prev = nil, nil
}

func (l *MsgQueue) PopTail() {
	if l.count == 0 {
		return
	}
	m := l.tail
	l.count--
	if l.count == 0 {
		l.tail, l.head = nil, nil
	} else {
		m.next.prev = nil
		l.tail = m.next
	}
	m.next, m.prev = nil, nil
}

// FragQueue tail -> x -> x -> head
type FragQueue struct {
	tail, head *Frag
	count      int
}

func (l *FragQueue) Reset() {
	l.count = 0
	l.tail = nil
	l.head = nil
}

func (l *FragQueue) Empty() bool {
	return l.count < 1
}

func (l *FragQueue) PushTail(m *Frag) {
	m.next = l.tail
	m.prev = nil
	if l.count == 0 {
		l.head = m
	} else {
		l.tail.prev = m
	}
	l.tail = m
	l.count++
}

func (l *FragQueue) PopHead() {
	if l.count == 0 {
		return
	}
	m := l.head
	l.count--
	if l.count == 0 {
		l.tail, l.head = nil, nil
	} else {
		m.prev.next = nil
		l.head = m.prev
	}
	m.next, m.prev = nil, nil
}
