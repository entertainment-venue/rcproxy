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
)

func TestFragQueue(t *testing.T) {
	fq := new(FragQueue)
	f1 := &Frag{Id: 1}
	f2 := &Frag{Id: 2}

	fq.PushTail(f1)
	assert.Same(t, f1, fq.tail)
	assert.Same(t, (*Frag)(nil), fq.tail.next)
	assert.Same(t, f1, fq.head)
	assert.Same(t, (*Frag)(nil), fq.head.prev)

	fq.PushTail(f2)
	assert.Same(t, f2, fq.tail)
	assert.Same(t, f1, fq.tail.next)
	assert.Same(t, f1, fq.head)
	assert.Same(t, f2, fq.head.prev)

	fq.PopHead()
	assert.Same(t, f2, fq.tail)
	assert.Same(t, (*Frag)(nil), fq.tail.next)
	assert.Same(t, f2, fq.head)
	assert.Same(t, (*Frag)(nil), fq.head.prev)

	fq.PopHead()
	assert.Same(t, (*Frag)(nil), fq.tail)
	assert.Same(t, (*Frag)(nil), fq.head)
}

func TestMsgQueue(t *testing.T) {
	q := new(MsgQueue)
	q.PushTail(&Msg{Id: 1})
	q.PushTail(&Msg{Id: 2})

	msg := q.head
	q.PopHead()
	assert.Equal(t, uint64(1), msg.Id)

	msg = q.head
	q.PopHead()
	assert.Equal(t, uint64(2), msg.Id)

	msg = q.head
	q.PopHead()
	assert.Equal(t, nil, nil)

	q.PushTail(&Msg{Id: 1})
	q.PushTail(&Msg{Id: 2})

	msg = q.tail
	q.PopTail()
	assert.Equal(t, uint64(2), msg.Id)

	msg = q.tail
	q.PopTail()
	assert.Equal(t, uint64(1), msg.Id)

	msg = q.tail
	q.PopTail()
	assert.Equal(t, nil, nil)
}
