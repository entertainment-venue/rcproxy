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
	"bytes"
)

var (
	LFByte   = byte('\n')
	CRByte   = byte('\r')
	LFCRByte = []byte{'\r', '\n'}
	LFCRStr  = string(LFCRByte)
)

// Buffer this structure is used to assist within-place decoding and avoid additional copies
type Buffer struct {
	buf []byte
	r   int // next position to read
}

// buffer single-threaded service, only one message will be decoded at the same time,
// so a global variable is maintained here to avoid frequent memory requests
var buffer Buffer

func NewBuffer(bs []byte) *Buffer {
	buffer.r = 0

	if len(bs) == 0 {
		buffer.buf = nil
		return &buffer
	}

	buffer.buf = bs
	return &buffer
}

// Empty whether buffer is empty or not
func (b *Buffer) Empty() bool {
	return len(b.buf) < 1
}

// TotalSize total number of bytes
func (b *Buffer) TotalSize() int {
	return len(b.buf)
}

// ReadSize number of bytes that have been read
func (b *Buffer) ReadSize() int {
	return b.r
}

// ReadBuf bytes that have been read
func (b *Buffer) ReadBuf() []byte {
	return b.buf[0:b.r]
}

// leftSize number of remaining unread bytes
func (b *Buffer) leftSize() int {
	return len(b.buf) - b.r
}

// leftBuf remaining unread bytes
func (b *Buffer) leftBuf() []byte {
	return b.buf[b.r:]
}

// ReadN reads bytes with the given length from Buffer without moving "r" pointer,
func (b *Buffer) ReadN(n int) ([]byte, error) {
	if b.leftSize() < 1 {
		return nil, EmptyLine
	}
	if n > b.leftSize() {
		return nil, ShortLine
	}
	r := b.r
	b.r = b.r + n
	return b.buf[r:b.r], nil
}

// PeekN reads bytes with the given length from Buffer and moving "r" pointer,
func (b *Buffer) PeekN(n int) ([]byte, error) {
	if b.leftSize() < 1 {
		return nil, EmptyLine
	}
	if n > b.leftSize() {
		return nil, ShortLine
	}
	return b.buf[b.r : b.r+n], nil
}

// ReadLine reads a line of bytes from Buffer without moving "r" pointer,
func (b *Buffer) ReadLine() ([]byte, error) {
	if b.leftSize() < 1 {
		return nil, EmptyLine
	}
	idx := bytes.IndexByte(b.leftBuf(), LFByte)
	if idx == -1 {
		return nil, ErrLFNotFound
	}
	buf, err := b.ReadN(idx + 1)
	if err != nil {
		return nil, err
	}
	if idx < 2 {
		return nil, EmptyLine
	}
	if buf[idx-1] != CRByte {
		return nil, ErrCRNotFound
	}
	return buf[:len(buf)-2], nil
}

// PeekAll reads all bytes from Buffer without moving "r" pointer,
func (b *Buffer) PeekAll() []byte {
	return b.buf
}