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
	"io"
	"reflect"
	"strings"

	"rcproxy/core/pkg/redis"
)

type ScriptRunner struct {
	ret interface{}
}

func (r *ScriptRunner) Run(c redis.Conn, s *Scanner) error {
	for {
		if err := r.execCommand(c, s); err != nil {
			if err == io.EOF {
				return nil
			} else {
				return err
			}
		}
	}
}

func (r *ScriptRunner) execCommand(c redis.Conn, s *Scanner) error {
	var items []interface{}
	line := 0
	for {
		line = s.line
		items = s.ScanCommand()
		err := s.Err()

		if err != nil && err != io.EOF {
			return err
		}

		if len(items) > 0 {
			break
		} else if err == io.EOF {
			return io.EOF
		}
	}

	var cmd string
	if s, ok := items[0].(string); !ok {
		return fmt.Errorf("invalid Command at line %d", line)
	} else {
		cmd = strings.ToUpper(s)
	}

	switch cmd {
	case "RET":
		if len(items) != 2 {
			return fmt.Errorf("RET must has 1 arg at line %d", line)
		}

		if err := r.checkRet(items[1], r.ret); err != nil {
			return fmt.Errorf("RET check err at line %d: %v", line, err)
		}

	case "RET_LEN":
		if len(items) != 2 {
			return fmt.Errorf("RET_LEN must has 1 arg at line %d", line)
		}

		if expectedLen, ok := items[1].(int64); !ok {
			return fmt.Errorf("RET_LEN parses arg err at line %d, not number but %T", line, items[1])
		} else if err := r.checkRetLen(expectedLen); err != nil {
			return fmt.Errorf("RET_LEN check err at line %d: %v", line, err)
		}

	case "RET_PRINT":
		r.printRet()
	default:
		// redis command
		var err error
		r.ret, err = c.Do(cmd, items[1:]...)
		if err != nil {
			if v, ok := err.(redis.Error); ok {
				r.ret = string(v)
				return nil
			}
			return fmt.Errorf("Do redis %v err at line %d, %v", items, line, err)
		}
	}
	return nil
}

func (r *ScriptRunner) printRet() {
	switch v := r.ret.(type) {
	case int64:
		fmt.Printf("%d\n", v)
	case string:
		fmt.Printf("%s\n", v)
	case []byte:
		fmt.Printf("%s\n", string(v))
	case []interface{}:
		fmt.Printf("%v\n", v)
	case nil:
		fmt.Printf("nil\n")
	default:
		fmt.Printf("%v\n", v)
	}
}

func (r *ScriptRunner) checkRetLen(l int64) error {
	size := 0
	switch v := r.ret.(type) {
	case int64:
		return fmt.Errorf("RET_LEN can not checm integer type")
	case string:
		size = len(v)
	case []byte:
		size = len(v)
	case []interface{}:
		size = len(v)
	case nil:
		size = 0
	default:
		return fmt.Errorf("Invalid type %T for RET_LEN", v)
	}

	if int64(size) != l {
		return fmt.Errorf("RET_LEN err, expected %d, but got %d", l, size)
	}

	return nil
}

func formatExpected(expected interface{}) string {
	if s, ok := expected.(string); ok {
		return s
	}

	return fmt.Sprintf("%v", expected)
}

func (r *ScriptRunner) checkRet(expected interface{}, got interface{}) error {
	equal := false
	var err error
	switch v := got.(type) {
	case int64:
		if n, ok := expected.(int64); ok {
			equal = n == v
		}
	case string:
		equal = v == formatExpected(expected)
	case []byte:
		equal = string(v) == formatExpected(expected)
	case nil:
		if s, ok := expected.(string); ok {
			equal = s == "nil"
		}
	case []interface{}:
		if a, ok := expected.([]interface{}); ok && len(a) == len(v) {
			for i, _ := range v {
				if err = r.checkRet(a[i], v[i]); err != nil {
					break
				}
			}
			equal = err == nil
		}
	default:
		return fmt.Errorf("invalid type %T for RET", v)
	}

	if !equal {
		return fmt.Errorf("expected %s(%v), but got %s(%s)", reflect.TypeOf(expected), expected, reflect.TypeOf(got), fmt.Sprintf("%+v", got))
	}

	return nil
}
