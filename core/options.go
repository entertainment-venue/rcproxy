// Copyright (c) 2022 The rcproxy Authors
// Copyright (c) 2019 Andy Pan
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
	"time"
)

// Option is a function that will set up option.
type Option func(opts *Options)

func loadOptions(options ...Option) *Options {
	opts := new(Options)
	for _, option := range options {
		option(opts)
	}
	return opts
}

// TCPSocketOpt is the type of TCP socket options.
type TCPSocketOpt int

// Options are configurations for the gnet application.
type Options struct {
	// ================================== Options for only server-side ==================================

	// ============================= Options for both server-side and client-side =============================

	// ReadBufferCap is the maximum number of bytes that can be read from the peer when the readable event comes.
	// The default value is 64KB, it can either be reduced to avoid starving the subsequent connections or increased
	// to read more data from a socket.
	//
	// Note that ReadBufferCap will always be converted to the least power of two integer value greater than
	// or equal to its real amount.
	ReadBufferCap int

	// WriteBufferCap is the maximum number of bytes that a static outbound buffer can hold,
	// if the data exceeds this value, the overflow will be stored in the elastic linked list buffer.
	// The default value is 64KB.
	//
	// Note that WriteBufferCap will always be converted to the least power of two integer value greater than
	// or equal to its real amount.
	WriteBufferCap int

	// TCPKeepAlive sets up a duration for (SO_KEEPALIVE) socket option.
	TCPKeepAlive time.Duration

	// SocketRecvBuffer sets the maximum socket receive buffer in bytes.
	SocketRecvBuffer int

	// SocketSendBuffer sets the maximum socket send buffer in bytes.
	SocketSendBuffer int

	// ============================= Options for redis server =============================

	// RedisServers address of the redis nodes
	RedisServers string

	// RedisMsgMaxLength indicates the maximum allowed packet length.
	// If the maximum allowed packet length is exceeded, an error is reported
	RedisMsgMaxLength int

	// RedisConnectionTimeout timeout of rcproxy with redis (unit: ms)
	RedisConnectionTimeout int

	// RedisRequestTimeout maximum request timeout with redis, otherwise return an error to the client (unit: ms)
	RedisRequestTimeout int

	// RedisServerConnections maximum number of connections to each redis node, best practice value is 1
	RedisServerConnections int

	// RedisPasswd redis password
	RedisPasswd string

	// RedisPreconnect whether to initialize redis connections in advance
	RedisPreconnect bool

	// RedisSlowlogSlowerThan threshold of redis slow query
	RedisSlowlogSlowerThan int64
}

// WithTCPKeepAlive sets up the SO_KEEPALIVE socket option with duration.
func WithTCPKeepAlive(tcpKeepAlive time.Duration) Option {
	return func(opts *Options) {
		opts.TCPKeepAlive = tcpKeepAlive
	}
}

// WithSocketRecvBuffer sets the maximum socket receive buffer in bytes.
func WithSocketRecvBuffer(recvBuf int) Option {
	return func(opts *Options) {
		opts.SocketRecvBuffer = recvBuf
	}
}

// WithSocketSendBuffer sets the maximum socket send buffer in bytes.
func WithSocketSendBuffer(sendBuf int) Option {
	return func(opts *Options) {
		opts.SocketSendBuffer = sendBuf
	}
}

// WithRedisServers sets up redis address
func WithRedisServers(addrs string) Option {
	return func(opts *Options) {
		opts.RedisServers = addrs
	}
}

// WithRedisMsgMaxLength sets up the maximum allowed packet length.
// If the maximum allowed packet length is exceeded, an error is reported
func WithRedisMsgMaxLength(length int) Option {
	return func(opts *Options) {
		opts.RedisMsgMaxLength = length
	}
}

// WithRedisPasswd sets up redis password
func WithRedisPasswd(passwd string) Option {
	return func(opts *Options) {
		opts.RedisPasswd = passwd
	}
}

// WithRedisPreconnect whether to initialize redis connections in advance
func WithRedisPreconnect(preconnect bool) Option {
	return func(opts *Options) {
		opts.RedisPreconnect = preconnect
	}
}

// WithRedisConnectTimeout sets up connect timeout of rcproxy with redis (unit: ms)
func WithRedisConnectTimeout(num int) Option {
	return func(opts *Options) {
		opts.RedisConnectionTimeout = num
	}
}

// WithRedisRequestTimeout sets up maximum request timeout with redis, otherwise return an error to the client
func WithRedisRequestTimeout(timeout int) Option {
	return func(opts *Options) {
		opts.RedisRequestTimeout = timeout
	}
}

// WithRedisServerConnections sets up maximum number of connections to each redis node, best practice value is 1
func WithRedisServerConnections(num int) Option {
	return func(opts *Options) {
		opts.RedisServerConnections = num
	}
}

// WithSlowlogSlowerThan sets up threshold of redis slow query
func WithSlowlogSlowerThan(num int64) Option {
	return func(opts *Options) {
		opts.RedisSlowlogSlowerThan = num
	}
}
