// Copyright (c) 2022 The rcproxy Authors
// Copyright (c) 2019 Andy Pan
// Copyright (c) 2018 Joshua J Baker
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
	"fmt"
	"io"
	"net"
	"os"
	"time"

	"golang.org/x/sys/unix"

	"rcproxy/core/codec"
	gio "rcproxy/core/internal/io"
	"rcproxy/core/internal/netpoll"
	"rcproxy/core/internal/socket"
	"rcproxy/core/internal/toolkit"
	"rcproxy/core/pkg/buffer/elastic"
	"rcproxy/core/pkg/constant"
	"rcproxy/core/pkg/errors"
	"rcproxy/core/pkg/logging"
	bsPool "rcproxy/core/pkg/pool/byteslice"
)

type conn struct {
	localAddr      net.Addr                // local addr
	remoteAddr     net.Addr                // remote addr
	loop           *eventloop              // connected event-loop
	outboundBuffer *elastic.Buffer         // buffer for data that is eligible to be sent to the peer
	pollAttachment *netpoll.PollAttachment // connection attachment for poller
	inboundBuffer  elastic.RingBuffer      // buffer for leftover data from the peer
	buffer         []byte                  // buffer for the latest bytes
	fd             int                     // file descriptor

	inMsgQueue   *MsgQueue  // queue of read client messages
	inFragQueue  *FragQueue // queue of read redis messages
	outFragQueue *FragQueue // queue of redis messages to be written

	opened     bool             // connection opened event fired
	isSlave    bool             // whether redis slave node
	initStep   int8             // number of steps required for redis connection initialization
	initStatus InitializeStatus // redis connection initialization status
	connType   ConnType         // client or server
}

func newTCPConn(fd int, el *eventloop, localAddr, remoteAddr net.Addr, connType ConnType, status InitializeStatus, isSlave bool) (c *conn) {
	c = &conn{
		fd:         fd,
		loop:       el,
		localAddr:  localAddr,
		remoteAddr: remoteAddr,

		initStatus:   status,
		isSlave:      isSlave,
		connType:     connType,
		inMsgQueue:   &MsgQueue{},
		inFragQueue:  &FragQueue{},
		outFragQueue: &FragQueue{},
	}
	c.outboundBuffer, _ = elastic.New(el.engine.opts.WriteBufferCap)
	c.pollAttachment = netpoll.GetPollAttachment()
	c.pollAttachment.FD, c.pollAttachment.Callback = fd, c.handleEvents
	return
}

func (c *conn) releaseTCP() {
	c.opened = false
	c.buffer = nil
	if addr, ok := c.localAddr.(*net.TCPAddr); ok && c.localAddr != c.loop.ln.addr {
		bsPool.Put(addr.IP)
		if len(addr.Zone) > 0 {
			bsPool.Put(toolkit.StringToBytes(addr.Zone))
		}
	}
	if addr, ok := c.remoteAddr.(*net.TCPAddr); ok {
		bsPool.Put(addr.IP)
		if len(addr.Zone) > 0 {
			bsPool.Put(toolkit.StringToBytes(addr.Zone))
		}
	}
	c.localAddr = nil
	c.remoteAddr = nil
	c.inboundBuffer.Done()
	c.outboundBuffer.Release()
	netpoll.PutPollAttachment(c.pollAttachment)
	c.pollAttachment = nil

	c.initStep = -1
	c.initStatus = InitializeNone
	c.isSlave = false
	c.connType = ConnNone
	c.inMsgQueue = nil
	c.inFragQueue = nil
	c.outFragQueue = nil
}

func (c *conn) open(buf []byte) error {
	n, err := unix.Write(c.fd, buf)
	if err != nil && err == unix.EAGAIN {
		_, _ = c.outboundBuffer.Write(buf)
		return nil
	}

	if err == nil && n < len(buf) {
		_, _ = c.outboundBuffer.Write(buf[n:])
	}

	return err
}

func (c *conn) write(data []byte) (n int, err error) {
	n = len(data)
	// If there is pending data in outbound buffer, the current data ought to be appended to the outbound buffer
	// for maintaining the sequence of network packets.
	if !c.outboundBuffer.IsEmpty() {
		_, _ = c.outboundBuffer.Write(data)
		return
	}

	var sent int
	if sent, err = unix.Write(c.fd, data); err != nil {
		// A temporary error occurs, append the data to outbound buffer, writing it back to the peer in the next round.
		if err == unix.EAGAIN {
			_, _ = c.outboundBuffer.Write(data)
			err = c.loop.poller.ModReadWrite(c.pollAttachment)
			return
		}
		return -1, c.loop.closeConn(c, os.NewSyscallError("write", err), ConnErr)
	}
	// Failed to send all data back to the peer, buffer the leftover data for the next round.
	if sent < n {
		_, _ = c.outboundBuffer.Write(data[sent:])
		err = c.loop.poller.ModReadWrite(c.pollAttachment)
	}
	return
}

func (c *conn) sread() (f *Frag, err error) {
	if c.InitializeStatus() == Initializing {
		err = EngineGlobal.sCodec.InitializingDecode(c)
		if err != nil {
			return nil, err
		}
	}

	f, err = EngineGlobal.sCodec.Decode(c)
	if err != nil {
		return nil, err
	}

	if f.Owner == nil {
		return f, nil
	}
	if f.Peer == nil {
		logging.Errorf("[%df][%ds] unknown r.Peer", f.Id, c.fd)
		return f, nil
	}

	switch f.Type {
	case codec.RspMoved, codec.RspAsk:
		logging.Warnf("[%dm|%df][%dc|%ds] got res: %s", f.MsgId(), f.Id, f.OwnerFd(), c.fd, f.RspBodyString())
		return f, codec.MovedOrAsk
	}

	if f.Done {
		logging.Debugf("[%dm|%df][%dc|%ds] frag already done", f.MsgId(), f.Id, f.OwnerFd(), c.fd)
		return nil, codec.Continue
	}

	f.slowLogCheck(c)

	if EngineGlobal.sCodec.sizeTooLarge(len(f.RspBody)) {
		f.Error = codec.ErrMsgRspTooLarge
	}

	f.Peer.FragDoneNumber++
	if f.Error.NotNil() {
		msg := f.Peer
		msg.Error = f.Error
		msg.FragDoneNumber = len(msg.Body)
		msg.RspBody = append(msg.RspBody[:0], msg.Error.Bytes()...)
		msg.Done = true
		for _, v := range msg.Body {
			v.Done = true
		}
		return f, nil
	}

	switch f.Peer.Type {
	case codec.ReqMget:
		err = EngineGlobal.sCodec.MGet(f, c.fd)
	case codec.ReqMset:
		err = EngineGlobal.sCodec.MSet(f, c.fd)
	case codec.ReqDel:
		err = EngineGlobal.sCodec.Del(f, c.fd)
	default:
		err = EngineGlobal.sCodec.Default(f)
	}

	return f, err
}

func (c *conn) cread() (*Msg, error) {
	m, err := EngineGlobal.cCodec.Decode(c)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (c *conn) writev(bs [][]byte) (n int, err error) {
	for _, b := range bs {
		n += len(b)
	}

	// If there is pending data in outbound buffer, the current data ought to be appended to the outbound buffer
	// for maintaining the sequence of network packets.
	if !c.outboundBuffer.IsEmpty() {
		_, _ = c.outboundBuffer.Writev(bs)
		return
	}

	var sent int
	if sent, err = gio.Writev(c.fd, bs); err != nil {
		// A temporary error occurs, append the data to outbound buffer, writing it back to the peer in the next round.
		if err == unix.EAGAIN {
			_, _ = c.outboundBuffer.Writev(bs)
			err = c.loop.poller.ModReadWrite(c.pollAttachment)
			return
		}
		return -1, c.loop.closeConn(c, os.NewSyscallError("write", err), ConnErr)
	}
	// Failed to send all data back to the peer, buffer the leftover data for the next round.
	if sent < n {
		var pos int
		for i := range bs {
			bn := len(bs[i])
			if sent < bn {
				bs[i] = bs[i][sent:]
				pos = i
				break
			}
			sent -= bn
		}
		_, _ = c.outboundBuffer.Writev(bs[pos:])
		err = c.loop.poller.ModReadWrite(c.pollAttachment)
	}
	return
}

type asyncWriteHook struct {
	callback AsyncCallback
	data     []byte
}

func (c *conn) asyncWrite(itf interface{}) (err error) {
	if !c.opened {
		return nil
	}

	hook := itf.(*asyncWriteHook)
	_, err = c.write(hook.data)
	if hook.callback != nil {
		_ = hook.callback(c)
	}
	return
}

type asyncWritevHook struct {
	callback AsyncCallback
	data     [][]byte
}

func (c *conn) asyncWritev(itf interface{}) (err error) {
	if !c.opened {
		return nil
	}

	hook := itf.(*asyncWritevHook)
	_, err = c.writev(hook.data)
	if hook.callback != nil {
		_ = hook.callback(c)
	}
	return
}

func (c *conn) resetBuffer() {
	c.buffer = c.buffer[:0]
	c.inboundBuffer.Reset()
}

func (c *conn) handleWriteSignal(_ interface{}) error {
	if !c.opened {
		return nil
	}

	if c.outFragQueue == nil {
		logging.Errorf("[%ds] outFragQueue cannot be nil", c.fd)
		return nil
	}

	if c.outFragQueue.count < 1 {
		return nil
	}

	var bs = make([][]byte, c.outFragQueue.count)
	for c.outFragQueue.head != nil {
		head := c.outFragQueue.head
		c.dequeueOutFrag()
		c.enqueueInFrag(head)
		bs = append(bs, head.Req)
	}
	_, err := c.writev(bs)
	return err
}

func (c *conn) sendWriteSignal() error {
	return c.loop.poller.Trigger(c.handleWriteSignal, nil)
}

func (c *conn) writeClusterNodes(_ interface{}) error {
	if !c.opened {
		return nil
	}

	frag := FragPool.Get()
	frag.Req = append(frag.Req, constant.ReqClusterNodes...)

	c.EnqueueOutFrag(frag)
	return nil
}

// ================================== Non-concurrency-safe API's ==================================

func (c *conn) Read(p []byte) (n int, err error) {
	if c.inboundBuffer.IsEmpty() {
		n = copy(p, c.buffer)
		c.buffer = c.buffer[n:]
		if n == 0 && len(p) > 0 {
			err = io.EOF
		}
		return
	}
	n, _ = c.inboundBuffer.Read(p)
	if n == len(p) {
		return
	}
	m := copy(p[n:], c.buffer)
	n += m
	c.buffer = c.buffer[m:]
	return
}

func (c *conn) Next(n int) (buf []byte, err error) {
	inBufferLen := c.inboundBuffer.Buffered()
	if totalLen := inBufferLen + len(c.buffer); n > totalLen {
		return nil, io.ErrShortBuffer
	} else if n <= 0 {
		n = totalLen
	}
	if c.inboundBuffer.IsEmpty() {
		buf = c.buffer[:n]
		c.buffer = c.buffer[n:]
		return
	}
	head, tail := c.inboundBuffer.Peek(n)
	defer c.inboundBuffer.Discard(n) //nolint:errcheck
	if len(head) >= n {
		return head[:n], err
	}
	c.loop.cache.Reset()
	c.loop.cache.Write(head)
	c.loop.cache.Write(tail)
	if inBufferLen >= n {
		return c.loop.cache.Bytes(), err
	}

	remaining := n - inBufferLen
	c.loop.cache.Write(c.buffer[:remaining])
	c.buffer = c.buffer[remaining:]
	return c.loop.cache.Bytes(), err
}

func (c *conn) Peek(n int) (buf []byte, err error) {
	inBufferLen := c.inboundBuffer.Buffered()
	if totalLen := inBufferLen + len(c.buffer); n > totalLen {
		return nil, io.ErrShortBuffer
	} else if n <= 0 {
		n = totalLen
	}
	if c.inboundBuffer.IsEmpty() {
		return c.buffer[:n], err
	}
	head, tail := c.inboundBuffer.Peek(n)
	if len(head) >= n {
		return head[:n], err
	}
	c.loop.cache.Reset()
	c.loop.cache.Write(head)
	c.loop.cache.Write(tail)
	if inBufferLen >= n {
		return c.loop.cache.Bytes(), err
	}

	remaining := n - inBufferLen
	c.loop.cache.Write(c.buffer[:remaining])
	return c.loop.cache.Bytes(), err
}

func (c *conn) Discard(n int) (int, error) {
	inBufferLen := c.inboundBuffer.Buffered()
	tempBufferLen := len(c.buffer)
	if inBufferLen+tempBufferLen < n || n <= 0 {
		c.resetBuffer()
		return inBufferLen + tempBufferLen, nil
	}
	if c.inboundBuffer.IsEmpty() {
		c.buffer = c.buffer[n:]
		return n, nil
	}

	discarded, _ := c.inboundBuffer.Discard(n)
	if discarded < inBufferLen {
		return discarded, nil
	}

	remaining := n - inBufferLen
	c.buffer = c.buffer[remaining:]
	return n, nil
}

func (c *conn) Write(p []byte) (int, error) {
	return c.write(p)
}

func (c *conn) Writev(bs [][]byte) (int, error) {
	return c.writev(bs)
}

func (c *conn) ReadFrom(r io.Reader) (int64, error) {
	return c.outboundBuffer.ReadFrom(r)
}

func (c *conn) WriteTo(w io.Writer) (n int64, err error) {
	if !c.inboundBuffer.IsEmpty() {
		if n, err = c.inboundBuffer.WriteTo(w); err != nil {
			return
		}
	}
	var m int
	m, err = w.Write(c.buffer)
	n += int64(m)
	c.buffer = c.buffer[m:]
	return
}

func (c *conn) Flush() error {
	if c.outboundBuffer.IsEmpty() {
		return nil
	}

	return c.loop.write(c)
}

func (c *conn) InboundBuffered() int {
	return c.inboundBuffer.Buffered() + len(c.buffer)
}

func (c *conn) OutboundBuffered() int {
	return c.outboundBuffer.Buffered()
}

func (c *conn) SetDeadline(_ time.Time) error {
	return errors.ErrUnsupportedOp
}

func (c *conn) SetReadDeadline(_ time.Time) error {
	return errors.ErrUnsupportedOp
}

func (c *conn) SetWriteDeadline(_ time.Time) error {
	return errors.ErrUnsupportedOp
}

func (c *conn) EnqueueInMsg(msg *Msg) {
	c.inMsgQueue.PushTail(msg)
}

func (c *conn) enqueueInFrag(frag *Frag) {
	c.inFragQueue.PushTail(frag)
	pushToTimeoutQueue(frag, c.loop.engine.opts.RedisRequestTimeout)
}

func (c *conn) EnqueueOutFrag(f *Frag) {
	c.outFragQueue.PushTail(f)
	logging.Debugfunc(func() string { return fmt.Sprintf("[%dm|%df][%dc|%ds] frag enqueue: %s", f.MsgId(), f.Id, f.OwnerFd(), c.fd, f.ReqString()) })

	if err := c.sendWriteSignal(); err != nil {
		logging.Errorf("[%dm|%df][%dc|%ds] failed to send write signal, err: %s", f.MsgId(), f.Id, f.OwnerFd(), c.fd, err)
		return
	}
}

func (c *conn) DequeueInFrag() *Frag {
	f := c.inFragQueue.head
	if f == nil {
		return nil
	}
	c.inFragQueue.PopHead()
	deleteFromTimeoutQueue(f)
	return f
}

func (c *conn) dequeueOutFrag() {
	c.outFragQueue.PopHead()
}

func (c *conn) dequeueInMsg() *Msg {
	if c.inMsgQueue.Empty() {
		return nil
	}
	head := c.inMsgQueue.head
	c.inMsgQueue.PopHead()
	return head
}

func (c *conn) WriteClusterNodes() error {
	return c.loop.poller.Trigger(c.writeClusterNodes, nil)
}

func (c *conn) AsyncWrite(buf []byte, callback AsyncCallback) error {
	return c.loop.poller.Trigger(c.asyncWrite, &asyncWriteHook{callback, buf})
}

func (c *conn) AsyncWritev(bs [][]byte, callback AsyncCallback) error {
	return c.loop.poller.Trigger(c.asyncWritev, &asyncWritevHook{callback, bs})
}

// Implementation of Socket interface

func (c *conn) Fd() int                        { return c.fd }
func (c *conn) Dup() (fd int, err error)       { fd, _, err = netpoll.Dup(c.fd); return }
func (c *conn) SetReadBuffer(bytes int) error  { return socket.SetRecvBuffer(c.fd, bytes) }
func (c *conn) SetWriteBuffer(bytes int) error { return socket.SetSendBuffer(c.fd, bytes) }
func (c *conn) SetLinger(sec int) error        { return socket.SetLinger(c.fd, sec) }
func (c *conn) SetKeepAlivePeriod(d time.Duration) error {
	return socket.SetKeepAlivePeriod(c.fd, int(d.Seconds()))
}
func (c *conn) ConnType() ConnType { return c.connType }
func (c *conn) IsOpened() bool     { return c.opened }

func (c *conn) IsSlave() bool     { return c.isSlave }
func (c *conn) SetIsSlave(b bool) { c.isSlave = b }

func (c *conn) InitializeStatus() InitializeStatus          { return c.initStatus }
func (c *conn) SetInitializeStatus(status InitializeStatus) { c.initStatus = status }
func (c *conn) InitializeStep() int8                        { return c.initStep }
func (c *conn) SetInitializeStep(step int8)                 { c.initStep = step }

func (c *conn) LocalAddr() string {
	if c.localAddr == nil {
		return "-"
	}
	return c.localAddr.String()
}
func (c *conn) RemoteAddr() string {
	if c.remoteAddr == nil {
		return "-"
	}
	return c.remoteAddr.String()
}

// ==================================== Concurrency-safe API's ====================================

func (c *conn) CloseWithCallback(callback AsyncCallback) error {
	return c.loop.poller.Trigger(func(_ interface{}) (err error) {
		err = c.loop.closeConn(c, nil, ConnEof)
		if callback != nil {
			_ = callback(c)
		}
		return
	}, nil)
}

func (c *conn) Close() error {
	return c.loop.poller.Trigger(func(_ interface{}) (err error) {
		err = c.loop.closeConn(c, nil, ConnEof)
		return
	}, nil)
}
