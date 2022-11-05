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

func TestActiveList(t *testing.T) {
	l := new(activeList)
	pc1 := &poolConn{}
	l.pushFront(pc1)
	assert.Equal(t, 1, l.count)
	assert.Same(t, pc1, l.front)
	assert.Same(t, pc1, l.back)

	pc2 := &poolConn{}
	l.pushFront(pc2)
	assert.Equal(t, 2, l.count)
	assert.Same(t, pc2, l.front)
	assert.Same(t, pc1, l.back)

	pc3 := &poolConn{}
	l.pushFront(pc3)
	assert.Equal(t, 3, l.count)
	assert.Same(t, pc3, l.front)
	assert.Same(t, pc1, l.back)

	l.popBack()
	assert.Equal(t, 2, l.count)
	assert.Same(t, pc3, l.front)
	assert.Same(t, pc2, l.back)

	l.popBack()
	assert.Equal(t, 1, l.count)
	assert.Same(t, pc3, l.front)
	assert.Same(t, pc3, l.back)
}
