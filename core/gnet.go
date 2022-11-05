// Copyright (c) 2022 The rcproxy Authors
// Copyright (c) 2019 Andy Pan
// Copyright (c) 2018 Joshua J Baker
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
	"context"
	"io"
	"strings"
	"sync"
	"time"

	"rcproxy/core/pkg/constant"
	"rcproxy/core/pkg/errors"
)

var EngineGlobal *Engine

// Action is an action that occurs after the completion of an event.
type Action int

const (
	// None indicates that no action should occur following an event.
	None Action = iota

	// Close closes the connection.
	Close

	// Shutdown shutdowns the engine.
	Shutdown
)

// ConnType enumeration values for client or redis
type ConnType byte

const (
	// ConnNone unknown conn
	ConnNone ConnType = ' '
	// ConnClient client conn
	ConnClient ConnType = 'c'
	// ConnServer redis conn
	ConnServer ConnType = 's'
)

// the initialization status of the redis connection
type InitializeStatus int8

const (
	// InitializeNone uninitialized
	InitializeNone InitializeStatus = -1
	// Initializing
	Initializing InitializeStatus = 0
	// Initialized
	Initialized InitializeStatus = 1
)

// Mapping of slots to redis nodes
type slotReplicaset [constant.RedisClusterSlots]*replicaset

func (sr *slotReplicaset) Set(slot int32, replicaset *replicaset) {
	sr[slot] = replicaset
}

func (sr *slotReplicaset) Get(slot int32) *replicaset {
	return sr[slot]
}

func (sr *slotReplicaset) NotExist(slot int32) bool {
	if sr[slot] == nil {
		return true
	}
	return false
}

func (sr *slotReplicaset) Reset() {
	for i := 0; i < constant.RedisClusterSlots; i++ {
		sr[i] = nil
	}
}

// Engine represents an engine context which provides some functions.
type Engine struct {
	// eng is the internal engine struct.
	eng *engine

	// cCodec decoder with client
	cCodec CRespCodec

	// sCodec decoder with redis
	sCodec SRespCodec

	// ProxyPool mapping of redis nodes to connection pools
	ProxyPool map[string]*Pool

	// ProxyAddrs slices of redis nodes
	ProxyAddrs []string

	// clusterChan cluster info is processed asynchronously,
	// there is no need to use the resources of the main thread
	clusterChan chan []byte

	// ClusterNodes redis cluster synchronization
	ClusterNodes ClusterNodes

	// Slots2Node mapping of slots to redis nodes
	Slots2Node slotReplicaset
}

// CountConnections counts the number of currently active connections and returns it.
func (s Engine) CountConnections() (count int) {
	return int(s.eng.el.loadCConn()) + int(s.eng.el.loadSConn())
}

// Reader is an interface that consists of a number of methods for reading that Conn must implement.
type Reader interface {
	// ================================== Non-concurrency-safe API's ==================================

	io.Reader
	io.WriterTo // must be non-blocking, otherwise it may block the event-loop.

	// Next returns a slice containing the next n bytes from the buffer,
	// advancing the buffer as if the bytes had been returned by Read.
	// If there are fewer than n bytes in the buffer, Next returns the entire buffer.
	// The error is ErrBufferFull if n is larger than b's buffer size.
	//
	// Note that the []byte buf returned by Next() is not allowed to be passed to a new goroutine,
	// as this []byte will be reused within event-loop.
	// If you have to use buf in a new goroutine, then you need to make a copy of buf and pass this copy
	// to that new goroutine.
	Next(n int) (buf []byte, err error)

	// Peek returns the next n bytes without advancing the reader. The bytes stop
	// being valid at the next read call. If Peek returns fewer than n bytes, it
	// also returns an error explaining why the read is short. The error is
	// ErrBufferFull if n is larger than b's buffer size.
	//
	// Note that the []byte buf returned by Peek() is not allowed to be passed to a new goroutine,
	// as this []byte will be reused within event-loop.
	// If you have to use buf in a new goroutine, then you need to make a copy of buf and pass this copy
	// to that new goroutine.
	Peek(n int) (buf []byte, err error)

	// Discard skips the next n bytes, returning the number of bytes discarded.
	//
	// If Discard skips fewer than n bytes, it also returns an error.
	// If 0 <= n <= b.Buffered(), Discard is guaranteed to succeed without
	// reading from the underlying io.Reader.
	Discard(n int) (discarded int, err error)

	// InboundBuffered returns the number of bytes that can be read from the current buffer.
	InboundBuffered() (n int)
}

// Writer is an interface that consists of a number of methods for writing that Conn must implement.
type Writer interface {
	// ================================== Non-concurrency-safe API's ==================================

	io.Writer
	io.ReaderFrom // must be non-blocking, otherwise it may block the event-loop.

	// Writev writes multiple byte slices to peer synchronously, you must call it in the current goroutine.
	Writev(bs [][]byte) (n int, err error)

	// Flush writes any buffered data to the underlying connection, you must call it in the current goroutine.
	Flush() (err error)

	// OutboundBuffered returns the number of bytes that can be read from the current buffer.
	OutboundBuffered() (n int)

	// ==================================== Concurrency-safe API's ====================================

	// AsyncWrite writes one byte slice to peer asynchronously, usually you would call it in individual goroutines
	// instead of the event-loop goroutines.
	AsyncWrite(buf []byte, callback AsyncCallback) (err error)

	// AsyncWritev writes multiple byte slices to peer asynchronously, usually you would call it in individual goroutines
	// instead of the event-loop goroutines.
	AsyncWritev(bs [][]byte, callback AsyncCallback) (err error)
}

// AsyncCallback is a callback which will be invoked after the asynchronous functions has finished executing.
type AsyncCallback func(c Conn) error

// Socket is a set of functions which manipulate the underlying file descriptor of a connection.
type Socket interface {
	// Fd returns the underlying file descriptor.
	Fd() int

	// Dup returns a copy of the underlying file descriptor.
	// It is the caller's responsibility to close fd when finished.
	// Closing c does not affect fd, and closing fd does not affect c.
	//
	// The returned file descriptor is different from the
	// connection's. Attempting to change properties of the original
	// using this duplicate may or may not have the desired effect.
	Dup() (int, error)

	// SetReadBuffer sets the size of the operating system's
	// receive buffer associated with the connection.
	SetReadBuffer(bytes int) error

	// SetWriteBuffer sets the size of the operating system's
	// transmit buffer associated with the connection.
	SetWriteBuffer(bytes int) error

	// IsOpened whether the connection is open
	IsOpened() bool

	// SetLinger sets the behavior of Close on a connection which still
	// has data waiting to be sent or to be acknowledged.
	//
	// If sec < 0 (the default), the operating system finishes sending the
	// data in the background.
	//
	// If sec == 0, the operating system discards any unsent or
	// unacknowledged data.
	//
	// If sec > 0, the data is sent in the background as with sec < 0. On
	// some operating systems after sec seconds have elapsed any remaining
	// unsent data may be discarded.
	SetLinger(sec int) error

	// SetKeepAlivePeriod tells operating system to send keep-alive messages on the connection
	// and sets period between TCP keep-alive probes.
	SetKeepAlivePeriod(d time.Duration) error
}

// Conn is an interface of underlying connection.
type Conn interface {
	Reader
	Writer
	Socket

	// ================================== Non-concurrency-safe API's ==================================

	// LocalAddr is the connection's local socket address.
	LocalAddr() (addr string)

	// RemoteAddr is the connection's remote peer address.
	RemoteAddr() (addr string)

	// SetDeadline implements net.Conn.
	SetDeadline(t time.Time) (err error)

	// SetReadDeadline implements net.Conn.
	SetReadDeadline(t time.Time) (err error)

	// SetWriteDeadline implements net.Conn.
	SetWriteDeadline(t time.Time) (err error)

	// ==================================== Concurrency-safe API's ====================================

	// CloseWithCallback closes the current connection, usually you don't need to pass a non-nil callback
	// because you should use OnClose() instead, the callback here is only for compatibility.
	CloseWithCallback(callback AsyncCallback) (err error)

	// Close closes the current connection, implements net.Conn.
	Close() (err error)
}

// CConn is an interface of client connection.
type CConn interface {
	Conn

	EnqueueInMsg(msg *Msg)
}

// SConn is an interface of redis server connection.
type SConn interface {
	Conn

	InitializeStatus() InitializeStatus
	SetInitializeStatus(_ InitializeStatus)

	InitializeStep() int8
	SetInitializeStep(_ int8)

	IsSlave() bool
	SetIsSlave(bool)

	EnqueueOutFrag(frag *Frag)
	DequeueInFrag() *Frag

	WriteClusterNodes() error
}

type (
	// EventHandler represents the engine events' callbacks for the Run call.
	// Each event has an Action return value that is used manage the state
	// of the connection and engine.
	EventHandler interface {
		// OnBoot fires when the engine is ready for accepting connections.
		// The parameter engine has information and various utilities.
		OnBoot(eng Engine) (action Action)

		// OnShutdown fires when the engine is being shut down, it is called right after
		// all event-loops and connections are closed.
		OnShutdown(eng Engine)

		// OnCOpened fires when a new client connection has been opened.
		OnCOpened(c CConn) (out []byte, action Action)

		// OnSOpened fires when a new redis server connection has been opened.
		OnSOpened(c SConn) (out []byte, action Action)

		// OnCClosed fires when a client connection has been closed.
		OnCClosed(c CConn, err error)

		// OnSClosed fires when a redis server connection has been closed.
		OnSClosed(c SConn, err error)

		// OnCReact fires when a client socket receives data from the peer.
		OnCReact(packet *Msg, c CConn) (out []byte, action Action)

		// OnMoved fires when a redis connection return moved/ask error
		OnMoved(addr string, slot int32, c SConn, f *Frag)

		// OnTicker fires every second for cluster nodes loop
		OnTicker()
	}

	// BuiltinEventEngine is a built-in implementation of EventHandler which sets up each method with a default implementation,
	// you can compose it with your own implementation of EventHandler when you don't want to implement all methods
	// in EventHandler.
	BuiltinEventEngine struct{}
)

// OnBoot fires when the engine is ready for accepting connections.
// The parameter engine has information and various utilities.
func (es *BuiltinEventEngine) OnBoot(_ Engine) (_ Action) {
	return
}

// OnShutdown fires when the engine is being shut down, it is called right after
// all event-loops and connections are closed.
func (es *BuiltinEventEngine) OnShutdown(_ Engine) {
}

// OnCOpened fires when a new client connection has been opened.
// The parameter out is the return value which is going to be sent back to the peer.
func (es *BuiltinEventEngine) OnCOpened(_ CConn) (_ []byte, _ Action) {
	return
}

// OnSOpened fires when a new redis server connection has been opened.
// The parameter out is the return value which is going to be sent back to the peer.
func (es *BuiltinEventEngine) OnSOpened(_ SConn) (_ []byte, _ Action) {
	return
}

// OnCClosed fires when a client connection has been closed.
// The parameter err is the last known connection error.
func (es *BuiltinEventEngine) OnCClosed(_ CConn, _ error) {
	return
}

// OnSClosed fires when a redis server connection has been closed.
// The parameter err is the last known connection error.
func (es *BuiltinEventEngine) OnSClosed(_ SConn, _ error) {
	return
}

// OnCReact fires when a client socket receives data from the peer.
func (es *BuiltinEventEngine) OnCReact(_ *Msg, _ CConn) (_ []byte, _ Action) {
	return
}

// OnMoved fires when a redis connection return moved/ask error
func (es *BuiltinEventEngine) OnMoved(_ string, _ int32, _ SConn, _ *Frag) {
}

// OnTicker fires every second for cluster nodes loop
func (es *BuiltinEventEngine) OnTicker() {
	return
}

// MaxStreamBufferCap is the default buffer size for each stream-oriented connection(TCP/Unix).
var MaxStreamBufferCap = 64 * 1024 // 64KB

// Run starts handling events on the specified address.
//
// Address should use a scheme prefix and be formatted
// like `tcp://192.168.0.10:9851`
// Valid network schemes:
//  tcp   - bind to both IPv4 and IPv6
//  tcp4  - IPv4
//  tcp6  - IPv6
//
// The "tcp" network scheme is assumed when one is not specified.
func Run(eventHandler EventHandler, protoAddr string, opts ...Option) (err error) {
	options := loadOptions(opts...)
	options.ReadBufferCap = MaxStreamBufferCap
	options.WriteBufferCap = MaxStreamBufferCap

	if options.RedisMsgMaxLength < 1 {
		options.RedisMsgMaxLength = 6 * 1024 * 1024
	}
	if options.RedisServerConnections < 1 {
		options.RedisServerConnections = 1
	}
	if options.RedisConnectionTimeout < 1 {
		options.RedisConnectionTimeout = 200
	}

	network, addr := parseProtoAddr(protoAddr)

	var ln *listener
	if ln, err = initListener(network, addr, options); err != nil {
		return
	}
	defer ln.close()

	return serve(eventHandler, ln, options, protoAddr)
}

var (
	allEngines sync.Map

	// shutdownPollInterval is how often we poll to check whether engine has been shut down during gnet.Stop().
	shutdownPollInterval = 500 * time.Millisecond
)

// Stop gracefully shuts down the engine without interrupting any active event-loops,
// it waits indefinitely for connections and event-loops to be closed and then shuts down.
func Stop(ctx context.Context, protoAddr string) error {
	var eng *engine
	if s, ok := allEngines.Load(protoAddr); ok {
		eng = s.(*engine)
		eng.signalShutdown()
		defer allEngines.Delete(protoAddr)
	} else {
		return errors.ErrEngineInShutdown
	}

	if eng.isInShutdown() {
		return errors.ErrEngineInShutdown
	}

	ticker := time.NewTicker(shutdownPollInterval)
	defer ticker.Stop()
	for {
		if eng.isInShutdown() {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func parseProtoAddr(addr string) (network, address string) {
	network = "tcp"
	address = strings.ToLower(addr)
	if strings.Contains(address, "://") {
		pair := strings.Split(address, "://")
		network = pair[0]
		address = pair[1]
	}
	return
}
