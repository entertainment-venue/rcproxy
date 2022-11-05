// Copyright (c) 2022 The rcproxy Authors
// Copyright (c) 2012 Gary Burd
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

package redis

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// Error represents an error returned in a command reply.
type Error string

func (err Error) Error() string { return string(err) }

type Conn interface {
	Info() (*Info, error)
	Do(cmd string, args ...interface{}) (interface{}, error)
	Send(cmd string, args ...interface{}) error
	Flush() error
	Receive() (interface{}, error)
	Close() error
}

// conn is the low-level implementation of Conn
type conn struct {
	err  error
	conn net.Conn

	pending int

	// Read
	readTimeout time.Duration
	br          *bufio.Reader

	// Write
	writeTimeout time.Duration
	bw           *bufio.Writer

	// Scratch space for formatting argument length.
	// '*' or '$', length, "\r\n"
	lenScratch [32]byte

	// Scratch space for formatting integers and floats.
	numScratch [40]byte
}

type Info struct {
	Version          string
	Loading          bool
	MasterLinkStatus string
}

// DialOption specifies an option for dialing a Redis server.
type DialOption struct {
	f func(*dialOptions)
}

type dialOptions struct {
	readTimeout  time.Duration
	writeTimeout time.Duration
	dialer       *net.Dialer
}

// DialReadTimeout specifies the timeout for reading a single command reply.
func DialReadTimeout(d time.Duration) DialOption {
	return DialOption{func(do *dialOptions) {
		do.readTimeout = d
	}}
}

// DialWriteTimeout specifies the timeout for writing a single command.
func DialWriteTimeout(d time.Duration) DialOption {
	return DialOption{func(do *dialOptions) {
		do.writeTimeout = d
	}}
}

// DialContext connects to the Redis server at the given network and
// address using the specified options and context.
func Dial(address, passwd string, options ...DialOption) (Conn, error) {
	do := dialOptions{
		dialer: &net.Dialer{
			Timeout:   time.Second * 30,
			KeepAlive: time.Minute * 5,
		},
		readTimeout:  3 * time.Second,
		writeTimeout: 3 * time.Second,
	}

	for _, option := range options {
		option.f(&do)
	}

	netConn, err := do.dialer.DialContext(context.Background(), "tcp", address)
	if err != nil {
		return nil, err
	}

	c := &conn{
		conn:         netConn,
		bw:           bufio.NewWriterSize(netConn, 4096*10),
		br:           bufio.NewReaderSize(netConn, 4096*10),
		readTimeout:  do.readTimeout,
		writeTimeout: do.writeTimeout,
	}

	if passwd != "" {
		if _, err := c.Do("AUTH", passwd); err != nil {
			netConn.Close()
			return nil, err
		}
	}

	return c, nil
}

func (c *conn) Close() error {
	err := c.err
	if c.err == nil {
		c.err = errors.New("redigo: closed")
		err = c.conn.Close()
	}
	return err
}

func (c *conn) fatal(err error) error {
	if c.err == nil {
		c.err = err
		// Close connection to force errors on subsequent calls and to unblock
		// other reader or writer.
		c.conn.Close()
	}
	return err
}

func (c *conn) Err() error {
	err := c.err
	return err
}

func (c *conn) writeLen(prefix byte, n int) error {
	c.lenScratch[len(c.lenScratch)-1] = '\n'
	c.lenScratch[len(c.lenScratch)-2] = '\r'
	i := len(c.lenScratch) - 3
	for {
		c.lenScratch[i] = byte('0' + n%10)
		i -= 1
		n = n / 10
		if n == 0 {
			break
		}
	}
	c.lenScratch[i] = prefix
	_, err := c.bw.Write(c.lenScratch[i:])
	return err
}

func (c *conn) writeString(s string) error {
	if err := c.writeLen('$', len(s)); err != nil {
		return err
	}
	if _, err := c.bw.WriteString(s); err != nil {
		return err
	}
	_, err := c.bw.WriteString("\r\n")
	return err
}

func (c *conn) writeBytes(p []byte) error {
	if err := c.writeLen('$', len(p)); err != nil {
		return err
	}
	if _, err := c.bw.Write(p); err != nil {
		return err
	}
	_, err := c.bw.WriteString("\r\n")
	return err
}

func (c *conn) writeInt64(n int64) error {
	return c.writeBytes(strconv.AppendInt(c.numScratch[:0], n, 10))
}

func (c *conn) writeFloat64(n float64) error {
	return c.writeBytes(strconv.AppendFloat(c.numScratch[:0], n, 'g', -1, 64))
}

func (c *conn) writeCommand(cmd string, args []interface{}) error {
	if err := c.writeLen('*', 1+len(args)); err != nil {
		return err
	}
	if err := c.writeString(cmd); err != nil {
		return err
	}
	for _, arg := range args {
		if err := c.writeArg(arg, true); err != nil {
			return err
		}
	}
	return nil
}

func (c *conn) writeArg(arg interface{}, argumentTypeOK bool) (err error) {
	switch arg := arg.(type) {
	case string:
		return c.writeString(arg)
	case []byte:
		return c.writeBytes(arg)
	case int:
		return c.writeInt64(int64(arg))
	case int64:
		return c.writeInt64(arg)
	case float64:
		return c.writeFloat64(arg)
	case bool:
		if arg {
			return c.writeString("1")
		} else {
			return c.writeString("0")
		}
	case nil:
		return c.writeString("")
	case Argument:
		if argumentTypeOK {
			return c.writeArg(arg.RedisArg(), false)
		}
		// See comment in default clause below.
		var buf bytes.Buffer
		fmt.Fprint(&buf, arg)
		return c.writeBytes(buf.Bytes())
	default:
		// This default clause is intended to handle builtin numeric types.
		// The function should return an error for other types, but this is not
		// done for compatibility with previous versions of the package.
		var buf bytes.Buffer
		fmt.Fprint(&buf, arg)
		return c.writeBytes(buf.Bytes())
	}
}

type protocolError string

func (pe protocolError) Error() string {
	return fmt.Sprintf("redigo: %s (possible server error or unsupported concurrent read by application)", string(pe))
}

// readLine reads a line of input from the RESP stream.
func (c *conn) readLine() ([]byte, error) {
	// To avoid allocations, attempt to read the line using ReadSlice. This call typically succeeds. The known case where the call fails is when reading the output from the MONITOR command.
	p, err := c.br.ReadSlice('\n')
	if err == bufio.ErrBufferFull {
		// The line does not fit in the bufio.Reader's buffer. Fall back to
		// allocating a buffer for the line.
		buf := append([]byte{}, p...)
		for err == bufio.ErrBufferFull {
			p, err = c.br.ReadSlice('\n')
			buf = append(buf, p...)
		}
		p = buf
	}
	if err != nil {
		return nil, err
	}
	i := len(p) - 2
	if i < 0 || p[i] != '\r' {
		return nil, protocolError("bad response line terminator")
	}
	return p[:i], nil
}

// parseLen parses bulk string and array lengths.
func parseLen(p []byte) (int, error) {
	if len(p) == 0 {
		return -1, protocolError("malformed length")
	}

	if p[0] == '-' && len(p) == 2 && p[1] == '1' {
		// handle $-1 and $-1 null replies.
		return -1, nil
	}

	var n int
	for _, b := range p {
		n *= 10
		if b < '0' || b > '9' {
			return -1, protocolError("illegal bytes in length")
		}
		n += int(b - '0')
	}

	return n, nil
}

// parseInt parses an integer reply.
func parseInt(p []byte) (interface{}, error) {
	if len(p) == 0 {
		return 0, protocolError("malformed integer")
	}

	var negate bool
	if p[0] == '-' {
		negate = true
		p = p[1:]
		if len(p) == 0 {
			return 0, protocolError("malformed integer")
		}
	}

	var n int64
	for _, b := range p {
		n *= 10
		if b < '0' || b > '9' {
			return 0, protocolError("illegal bytes in length")
		}
		n += int64(b - '0')
	}

	if negate {
		n = -n
	}
	return n, nil
}

var (
	okReply   interface{} = "OK"
	pongReply interface{} = "PONG"
)

func (c *conn) readReply() (interface{}, error) {
	line, err := c.readLine()
	if err != nil {
		return nil, err
	}
	if len(line) == 0 {
		return nil, protocolError("short response line")
	}
	switch line[0] {
	case '+':
		switch string(line[1:]) {
		case "OK":
			// Avoid allocation for frequent "+OK" response.
			return okReply, nil
		case "PONG":
			// Avoid allocation in PING command benchmarks :)
			return pongReply, nil
		default:
			return string(line[1:]), nil
		}
	case '-':
		return Error(line[1:]), nil
	case ':':
		return parseInt(line[1:])
	case '$':
		n, err := parseLen(line[1:])
		if n < 0 || err != nil {
			return nil, err
		}
		p := make([]byte, n)
		_, err = io.ReadFull(c.br, p)
		if err != nil {
			return nil, err
		}
		if line, err := c.readLine(); err != nil {
			return nil, err
		} else if len(line) != 0 {
			return nil, protocolError("bad bulk string format")
		}
		return p, nil
	case '*':
		n, err := parseLen(line[1:])
		if n < 0 || err != nil {
			return nil, err
		}
		r := make([]interface{}, n)
		for i := range r {
			r[i], err = c.readReply()
			if err != nil {
				return nil, err
			}
		}
		return r, nil
	}
	return nil, protocolError("unexpected response line")
}

func (c *conn) Info() (*Info, error) {
	i, err := c.DoWithTimeout(c.readTimeout, "info")
	if err != nil {
		return nil, err
	}

	msg, ok := i.([]byte)
	if !ok {
		return nil, errors.New("redis info asset error")
	}

	if len(msg) < 1 {
		return nil, errors.New("redis info empty")
	}
	if msg[0] == '-' {
		return nil, errors.New("redis info error, ret msg: " + string(msg))
	}

	info := new(Info)
	n := bytes.IndexByte(msg, '\n')
	msgStr := string(msg[n+1:])
	lines := strings.Split(msgStr, "\r\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "loading:") {
			info.Loading = strings.TrimSpace(strings.TrimPrefix(line, "loading:")) != "0"
		}
		if strings.HasPrefix(line, "master_link_status:") {
			info.MasterLinkStatus = strings.TrimSpace(strings.TrimPrefix(line, "master_link_status:"))
		}
		if strings.HasPrefix(line, "redis_version:") {
			info.Version = strings.TrimSpace(strings.TrimPrefix(line, "redis_version:"))
		}
	}
	return info, nil
}

func (c *conn) Do(cmd string, args ...interface{}) (interface{}, error) {
	return c.DoWithTimeout(c.readTimeout, cmd, args...)
}

func (c *conn) Send(cmd string, args ...interface{}) error {
	c.pending += 1
	if c.writeTimeout != 0 {
		if err := c.conn.SetWriteDeadline(time.Now().Add(c.writeTimeout)); err != nil {
			return c.fatal(err)
		}
	}
	if err := c.writeCommand(cmd, args); err != nil {
		return c.fatal(err)
	}
	return nil
}

func (c *conn) Flush() error {
	if c.writeTimeout != 0 {
		if err := c.conn.SetWriteDeadline(time.Now().Add(c.writeTimeout)); err != nil {
			return c.fatal(err)
		}
	}
	if err := c.bw.Flush(); err != nil {
		return c.fatal(err)
	}
	return nil
}

func (c *conn) Receive() (interface{}, error) {
	return c.ReceiveWithTimeout(c.readTimeout)
}

func (c *conn) ReceiveContext(ctx context.Context) (interface{}, error) {
	var realTimeout time.Duration
	if dl, ok := ctx.Deadline(); ok {
		timeout := time.Until(dl)
		if timeout >= c.readTimeout && c.readTimeout != 0 {
			realTimeout = c.readTimeout
		} else if timeout <= 0 {
			return nil, c.fatal(context.DeadlineExceeded)
		} else {
			realTimeout = timeout
		}
	} else {
		realTimeout = c.readTimeout
	}
	endch := make(chan struct{})
	var r interface{}
	var e error
	go func() {
		defer close(endch)

		r, e = c.ReceiveWithTimeout(realTimeout)
	}()
	select {
	case <-ctx.Done():
		return nil, c.fatal(ctx.Err())
	case <-endch:
		return r, e
	}
}

func (c *conn) ReceiveWithTimeout(timeout time.Duration) (reply interface{}, err error) {
	var deadline time.Time
	if timeout != 0 {
		deadline = time.Now().Add(timeout)
	}
	if err := c.conn.SetReadDeadline(deadline); err != nil {
		return nil, c.fatal(err)
	}

	if reply, err = c.readReply(); err != nil {
		return nil, c.fatal(err)
	}
	// When using pub/sub, the number of receives can be greater than the
	// number of sends. To enable normal use of the connection after
	// unsubscribing from all channels, we do not decrement pending to a
	// negative value.
	//
	// The pending field is decremented after the reply is read to handle the
	// case where Receive is called before Send.
	if c.pending > 0 {
		c.pending -= 1
	}
	if err, ok := reply.(Error); ok {
		return nil, err
	}
	return
}

func (c *conn) DoWithTimeout(readTimeout time.Duration, cmd string, args ...interface{}) (interface{}, error) {

	pending := c.pending
	c.pending = 0

	if cmd == "" && pending == 0 {
		return nil, nil
	}

	if c.writeTimeout != 0 {
		if err := c.conn.SetWriteDeadline(time.Now().Add(c.writeTimeout)); err != nil {
			return nil, c.fatal(err)
		}
	}

	if err := c.writeCommand(cmd, args); err != nil {
		return nil, c.fatal(err)
	}

	if err := c.bw.Flush(); err != nil {
		return nil, c.fatal(err)
	}

	var deadline time.Time
	if readTimeout != 0 {
		deadline = time.Now().Add(readTimeout)
	}
	if err := c.conn.SetReadDeadline(deadline); err != nil {
		return nil, c.fatal(err)
	}

	var err error
	var reply interface{}
	for i := 0; i <= pending; i++ {
		var e error
		if reply, e = c.readReply(); e != nil {
			return nil, c.fatal(e)
		}
		if e, ok := reply.(Error); ok && err == nil {
			err = e
		}
	}

	return reply, err
}

// Argument is the interface implemented by an object which wants to control how
// the object is converted to Redis bulk strings.
type Argument interface {
	// RedisArg returns a value to be encoded as a bulk string per the
	// conversions listed in the section 'Executing Commands'.
	// Implementations should typically return a []byte or string.
	RedisArg() interface{}
}
