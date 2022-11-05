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
	"strconv"
	"unicode"
	"unicode/utf8"
)

type Scanner struct {
	src []byte // source

	// scanning state
	ch       rune // current character
	offset   int  // character offset
	rdOffset int  // reading offset (position after current character)

	line int // current line

	err error

	// items for one command
	items []interface{}
	// handle array type
	arrayItems [][]interface{}
}

const bom = 0xFEFF // byte order mark, only permitted as very first character

// Read the next Unicode char into s.ch.
// s.ch < 0 means end-of-file.
//
func (s *Scanner) next() {
	if s.rdOffset < len(s.src) {
		s.offset = s.rdOffset
		if s.ch == '\n' {
			s.line++
		}
		r, w := rune(s.src[s.rdOffset]), 1
		switch {
		case r == 0:
			s.error(s.offset, "illegal character NUL")
		case r >= 0x80:
			// not ASCII
			r, w = utf8.DecodeRune(s.src[s.rdOffset:])
			if r == utf8.RuneError && w == 1 {
				s.error(s.offset, "illegal UTF-8 encoding")
			} else if r == bom && s.offset > 0 {
				s.error(s.offset, "illegal byte order mark")
			}
		}
		s.rdOffset += w
		s.ch = r
	} else {
		s.offset = len(s.src)
		if s.ch == '\n' {
			s.line++
		}
		s.ch = -1 // eof
	}
}

func (s *Scanner) Init(src []byte) {
	s.src = src

	s.ch = ' '
	s.offset = 0
	s.rdOffset = 0
	s.line = 1

	s.next()
	if s.ch == bom {
		s.next() // ignore BOM at file beginning
	}
}

func (s *Scanner) error(offs int, msg string) {
	if s.err == nil {
		s.err = fmt.Errorf("An error occurs at line %d, offset %d, err: %v", s.line, offs, msg)
	}
}

func (s *Scanner) scanComment() string {
	offs := s.offset - 1

	s.next()
	for s.ch != '\n' && s.ch >= 0 {
		s.next()
	}

	lit := s.src[offs:s.offset]

	return string(lit)
}

func isLetter(ch rune) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_' || ch >= 0x80 && unicode.IsLetter(ch)
}

func isDigit(ch rune) bool {
	return '0' <= ch && ch <= '9' || ch >= 0x80 && unicode.IsDigit(ch)
}

func (s *Scanner) scanIdentifier() string {
	offs := s.offset
	for isLetter(s.ch) || isDigit(s.ch) {
		s.next()
	}
	return string(s.src[offs:s.offset])
}

func digitVal(ch rune) int {
	switch {
	case '0' <= ch && ch <= '9':
		return int(ch - '0')
	case 'a' <= ch && ch <= 'f':
		return int(ch - 'a' + 10)
	case 'A' <= ch && ch <= 'F':
		return int(ch - 'A' + 10)
	}
	return 16 // larger than any legal digit val
}

func (s *Scanner) scanMantissa(base int) {
	for digitVal(s.ch) < base {
		s.next()
	}
}

func (s *Scanner) scanNumber() interface{} {
	offs := s.offset

	isInteger := true

	if s.ch == '0' {
		// int or float
		offs := s.offset
		s.next()
		if s.ch == 'x' || s.ch == 'X' {
			// hexadecimal int
			s.next()
			s.scanMantissa(16)
			if s.offset-offs <= 2 {
				// only scanned "0x" or "0X"
				s.error(offs, "illegal hexadecimal number")
				return nil
			}
		} else {
			// octal int or float
			seenDecimalDigit := false
			s.scanMantissa(8)
			if s.ch == '8' || s.ch == '9' {
				// illegal octal int or float
				seenDecimalDigit = true
				s.scanMantissa(10)
			}
			if s.ch == '.' || s.ch == 'e' || s.ch == 'E' || s.ch == 'i' {
				goto fraction
			} else if seenDecimalDigit {
				// octal int
				s.error(offs, "illegal octal number")
				return nil
			}
		}
		goto exit
	}

	s.scanMantissa(10)

fraction:
	if s.ch == '.' {
		isInteger = false
		s.next()
		s.scanMantissa(10)
	}

	if s.ch == 'e' || s.ch == 'E' {
		isInteger = false
		s.next()
		if s.ch == '-' || s.ch == '+' {
			s.next()
		}
		s.scanMantissa(10)
	}

	if s.ch == 'i' {
		s.error(offs, fmt.Sprintf("illegal number, can not support image number"))
		return nil
	}

exit:
	var v interface{}
	var err error
	if isInteger {
		v, err = strconv.ParseInt(string(s.src[offs:s.offset]), 10, 64)
	} else {
		v, err = strconv.ParseFloat(string(s.src[offs:s.offset]), 64)
	}

	if err != nil {
		s.error(offs, fmt.Sprintf("illegal number, parse err: %v", err))
		return nil
	}

	return v
}

// scanEscape parses an escape sequence where rune is the accepted
// escaped quote. In case of a syntax error, it stops at the offending
// character (without consuming it) and returns false. Otherwise
// it returns true.
func (s *Scanner) scanEscape(quote rune) bool {
	offs := s.offset

	var n int
	var base, max uint32
	switch s.ch {
	case 'a', 'b', 'f', 'n', 'r', 't', 'v', '\\', quote:
		s.next()
		return true
	case '0', '1', '2', '3', '4', '5', '6', '7':
		n, base, max = 3, 8, 255
	case 'x':
		s.next()
		n, base, max = 2, 16, 255
	case 'u':
		s.next()
		n, base, max = 4, 16, unicode.MaxRune
	case 'U':
		s.next()
		n, base, max = 8, 16, unicode.MaxRune
	default:
		msg := "unknown escape sequence"
		if s.ch < 0 {
			msg = "escape sequence not terminated"
		}
		s.error(offs, msg)
		return false
	}

	var x uint32
	for n > 0 {
		d := uint32(digitVal(s.ch))
		if d >= base {
			msg := fmt.Sprintf("illegal character %#U in escape sequence", s.ch)
			if s.ch < 0 {
				msg = "escape sequence not terminated"
			}
			s.error(s.offset, msg)
			return false
		}
		x = x*base + d
		s.next()
		n--
	}

	if x > max || 0xD800 <= x && x < 0xE000 {
		s.error(offs, "escape sequence is invalid Unicode code point")
		return false
	}

	return true
}

func (s *Scanner) scanString() string {
	// '"' opening already consumed
	offs := s.offset - 1

	for {
		ch := s.ch
		if ch == '\n' || ch < 0 {
			s.error(offs, "string literal not terminated")
			break
		}
		s.next()
		if ch == '"' {
			break
		}
		if ch == '\\' {
			s.scanEscape('"')
		}
	}

	// remove quote
	return string(s.src[offs+1 : s.offset-1])
}

func (s *Scanner) skipWhitespace() {
	for s.ch == ' ' || s.ch == '\t' || s.ch == '\r' {
		s.next()
	}
}

func (s *Scanner) Err() error {
	return s.err
}

func (s *Scanner) inBracket() bool {
	return len(s.arrayItems) > 0
}

func (s *Scanner) ScanCommand() []interface{} {
	s.items = make([]interface{}, 0)
	s.arrayItems = make([][]interface{}, 0)

	s.scanCommand()
	return s.items
}

func (s *Scanner) scanCommand() {
	var v interface{}
	for {
		v = nil
		s.skipWhitespace()

		switch ch := s.ch; {
		case isLetter(ch):
			v = s.scanIdentifier()
		case '0' <= ch && ch <= '9':
			v = s.scanNumber()
		default:
			s.next()
			switch ch {
			case -1:
				// EOF
				s.err = io.EOF
				return
			case '\n':
				if len(s.items) > 0 {
					return
				}
			case '"':
				v = s.scanString()
			case '-':
				// we only care negative number
				if '0' <= s.ch && s.ch <= '9' {
					v = s.scanNumber()
					switch n := v.(type) {
					case int64:
						v = -n
					case float64:
						v = -n
					}
				} else {
					s.error(s.offset, "only support negative number now")
					return
				}
			case '[':
				s.arrayItems = append(s.arrayItems, make([]interface{}, 0))
			case ']':
				if len(s.arrayItems) == 0 {
					s.error(s.offset, "invalid ], no corresponding [")
					return
				}
				// pop last array
				n := len(s.arrayItems) - 1
				v = s.arrayItems[n]
				s.arrayItems = s.arrayItems[0:n]
			case '#':
				s.scanComment()
			case ',':
				if !s.inBracket() {
					s.error(s.offset, fmt.Sprintf(", must in bracket for array type"))
					return
				}
			default:
				s.error(s.offset, fmt.Sprintf("illegal character %#U", ch))
				return
			}
		}

		if s.err != nil {
			break
		}

		if v != nil {
			if s.inBracket() {
				n := len(s.arrayItems) - 1
				b := s.arrayItems[n]
				b = append(b, v)
				s.arrayItems[n] = b
			} else {
				s.items = append(s.items, v)
			}
		}
	}
	return
}
