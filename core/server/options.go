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

package server

type Option func(opts *Options)

func loadOptions(options ...Option) *Options {
	opts := new(Options)
	for _, option := range options {
		option(opts)
	}
	return opts
}

type Options struct {
	Password           string
	DisableSlave       bool
	ServerRetryTimeout int
}

func WithRedisPassword(passwd string) Option {
	return func(opts *Options) {
		opts.Password = passwd
	}
}

func WithServerRetryTimeout(timeout int) Option {
	return func(opts *Options) {
		opts.ServerRetryTimeout = timeout
	}
}

func WithDisableRedisSlave(disable bool) Option {
	return func(opts *Options) {
		opts.DisableSlave = disable
	}
}
