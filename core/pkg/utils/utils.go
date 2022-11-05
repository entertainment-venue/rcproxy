// Copyright (c) 2022 The rcproxy Authors
//
// Licensed under the Apache License, Version 2.0 (the "License"): you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package utils

import (
	"reflect"
	"unsafe"
)

func S2B(s string) []byte {
	sh := (*reflect.StringHeader)(unsafe.Pointer(&s))
	bh := reflect.SliceHeader{Data: sh.Data, Len: sh.Len, Cap: sh.Len}
	return *(*[]byte)(unsafe.Pointer(&bh))
}

func B2S(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

// FormatRedisRESPMessages formatting redis RESP messages
func FormatRedisRESPMessages(resp []byte) string {
	var bs = make([]byte, len(resp))
	for i, v := range resp {
		if v == '\r' || v == '\n' {
			bs[i] = '.'
			continue
		}
		bs[i] = v
	}
	return B2S(bs)
}
