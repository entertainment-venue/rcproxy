// Copyright (c) 2022 The rcproxy Authors
// Copyright (c) 2015 siddontang
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
//
// Use of this source code is governed by a MIT license that can be found
// at https://github.com/siddontang/redis-test/blob/master/LICENSE

package test

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"rcproxy/core/pkg/redis"
)

// Integration Testing

const ProxyAddr = "127.0.0.1:9736"

func TestOther(t *testing.T) {
	run(t, "other")
}

func TestString(t *testing.T) {
	run(t, "string")
}

func TestHash(t *testing.T) {
	run(t, "hash")
}

func TestList(t *testing.T) {
	run(t, "list")
}

func TestSet(t *testing.T) {
	run(t, "set")
}

func TestZset(t *testing.T) {
	run(t, "zset")
}

func TestEval(t *testing.T) {
	run(t, "eval")
}

func TestDial(t *testing.T) {
	conn, err := redis.Dial(ProxyAddr, "")
	if err != nil {
		t.Errorf("unknow redis addr")
		return
	}
	defer conn.Close()
}

func TestPipeline(t *testing.T) {
	conn, _ := redis.Dial(
		ProxyAddr,
		"",
		redis.DialReadTimeout(100*time.Second),
		redis.DialWriteTimeout(100*time.Second),
	)

	conn.Do("del", "a", "b", "c", "d", "e")
	conn.Do("set", "a", "1")
	conn.Do("lpush", "b", "2")
	conn.Do("lpush", "b", "3")
	conn.Do("lpush", "b", "4")
	conn.Do("lpush", "b", "5")
	conn.Do("set", "c", "6")
	conn.Do("mset", "d", "7", "e", "8")

	conn.Send("get", "a")
	conn.Send("lrange", "b", "1", "3")
	conn.Send("get", "c")
	conn.Send("get", "d")
	conn.Send("get", "e")
	conn.Send("mget", "a", "e")
	conn.Flush()
	a, _ := conn.Receive()
	assert.Equal(t, "1", string(a.([]byte)))
	a, _ = conn.Receive()
	l := a.([]interface{})
	assert.Equal(t, "4", string(l[0].([]byte)))
	assert.Equal(t, "3", string(l[1].([]byte)))
	assert.Equal(t, "2", string(l[2].([]byte)))
	a, _ = conn.Receive()
	assert.Equal(t, "6", string(a.([]byte)))
	a, _ = conn.Receive()
	assert.Equal(t, "7", string(a.([]byte)))
	a, _ = conn.Receive()
	assert.Equal(t, "8", string(a.([]byte)))
	a, _ = conn.Receive()
	l = a.([]interface{})
	assert.Equal(t, "1", string(l[0].([]byte)))
	assert.Equal(t, "8", string(l[1].([]byte)))
}

func run(t *testing.T, dsl string) {
	conn, err := redis.Dial(ProxyAddr, "")
	if err != nil {
		t.Errorf("unknow redis addr")
		return
	}
	defer conn.Close()

	runCase(t, conn, dsl)
}

func runCase(t *testing.T, conn redis.Conn, testCase string) {
	fileName := fmt.Sprintf("case/%s.dsl", testCase)
	f, err := os.Open(fileName)
	if err != nil {
		t.Errorf("unknown script %s err: %s", fileName, err)
		return
	}
	defer f.Close()

	data, err := ioutil.ReadAll(f)
	if err != nil {
		t.Errorf("Read file %s err: %s", fileName, err)
		return
	}

	s := &Scanner{}
	s.Init(data)

	r := &ScriptRunner{}
	err = r.Run(conn, s)
	if err != nil {
		t.Errorf("Run script %s err :%v\n", fileName, err)
		return
	}
}
