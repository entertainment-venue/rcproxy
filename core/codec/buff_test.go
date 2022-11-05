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
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_ReadN(t *testing.T) {
	b := NewBuffer([]byte{11, 15, 13, 10, 20, 29})
	n, err := b.ReadN(1)
	assert.Equal(t, []byte{11}, n)
	assert.Equal(t, nil, err)

	n, err = b.ReadN(2)
	assert.Equal(t, []byte{15, 13}, n)
	assert.Equal(t, nil, err)
	n, err = b.ReadN(10)
	assert.Equal(t, ShortLine, err)

	assert.Equal(t, 3, b.leftSize())
}

func Test_PeekN(t *testing.T) {
	b := NewBuffer([]byte{11, 15, 13, 10, 20, 29})

	n, err := b.PeekN(1)
	assert.Equal(t, []byte{11}, n)
	assert.Equal(t, nil, err)

	n, err = b.PeekN(3)
	assert.Equal(t, []byte{11, 15, 13}, n)
	assert.Equal(t, nil, err)

	assert.Equal(t, 6, b.leftSize())
}

func Test_ReadLine(t *testing.T) {
	b := NewBuffer([]byte{11, 15, 13, 10, 20, 11, 29, 13, 10})

	n, err := b.ReadLine()
	assert.Equal(t, []byte{11, 15}, n)
	assert.Equal(t, nil, err)
	assert.Equal(t, 5, b.leftSize())

	n, err = b.ReadLine()
	assert.Equal(t, []byte{20, 11, 29}, n)
	assert.Equal(t, nil, err)
	assert.Equal(t, 0, b.leftSize())

	n, err = b.ReadLine()
	assert.Equal(t, EmptyLine, err)
}
