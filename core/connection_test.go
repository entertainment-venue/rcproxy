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
	"io"
	"time"

	"github.com/stretchr/testify/mock"
)

type mockedConn struct {
	mock.Mock
}

func (_ *mockedConn) LocalAddr() (addr string)                                    { return "" }
func (_ *mockedConn) RemoteAddr() (addr string)                                   { return "" }
func (_ *mockedConn) ConnType() ConnType                                          { return ConnClient }
func (_ *mockedConn) IsOpened() bool                                              { return true }
func (_ *mockedConn) EnqueueInMsg(_ *Msg)                                         {}
func (_ *mockedConn) SetIsSlave(bool)                                             {}
func (_ *mockedConn) Discard(n int) (discarded int, err error)                    { return }
func (_ *mockedConn) InboundBuffered() (n int)                                    { return }
func (_ *mockedConn) Writev(bs [][]byte) (n int, err error)                       { return }
func (_ *mockedConn) Flush() (err error)                                          { return }
func (_ *mockedConn) Write(p []byte) (n int, err error)                           { return }
func (_ *mockedConn) OutboundBuffered() (n int)                                   { return }
func (_ *mockedConn) Dup() (int, error)                                           { return 0, nil }
func (_ *mockedConn) SetReadBuffer(bytes int) error                               { return nil }
func (_ *mockedConn) SetWriteBuffer(bytes int) error                              { return nil }
func (_ *mockedConn) SetLinger(sec int) error                                     { return nil }
func (_ *mockedConn) SetKeepAlivePeriod(d time.Duration) error                    { return nil }
func (_ *mockedConn) ReadFrom(r io.Reader) (n int64, err error)                   { return }
func (_ *mockedConn) WriteTo(w io.Writer) (n int64, err error)                    { return }
func (_ *mockedConn) Next(n int) (buf []byte, err error)                          { return }
func (_ *mockedConn) AsyncWritev(bs [][]byte, callback AsyncCallback) (err error) { return }
func (_ *mockedConn) AsyncWrite(buf []byte, callback AsyncCallback) (err error)   { return }
func (_ *mockedConn) WriteDelayed() error                                         { return nil }
func (_ *mockedConn) Close() error                                                { return nil }
func (_ *mockedConn) SetDeadline(t time.Time) (err error)                         { return nil }
func (_ *mockedConn) SetReadDeadline(t time.Time) (err error)                     { return nil }
func (_ *mockedConn) SetWriteDeadline(t time.Time) (err error)                    { return nil }
func (_ *mockedConn) ForceClose() error                                           { return nil }
func (_ *mockedConn) CloseWithCallback(callback AsyncCallback) (err error)        { return }
func (m *mockedConn) Read(p []byte) (n int, err error)                            { return }
func (_ *mockedConn) InitializeStatus() InitializeStatus                          { return Initialized }
func (_ *mockedConn) SetInitializeStatus(_ InitializeStatus)                      {}
func (_ *mockedConn) InitializeStep() int8                                        { return -1 }
func (_ *mockedConn) SetInitializeStep(_ int8)                                    {}
func (_ *mockedConn) IsSlave() bool                                               { return true }
func (_ *mockedConn) EnqueueOutFrag(_ *Frag)                                      {}
func (_ *mockedConn) WriteClusterNodes() error                                    { return nil }
func (m *mockedConn) Fd() int {
	return m.Called().Get(0).(int)
}
func (m *mockedConn) Peek(n int) (buf []byte, err error) {
	return m.Called().Get(0).([]byte), nil
}
func (m *mockedConn) DequeueInFrag() *Frag {
	return m.Called().Get(0).(*Frag)
}
