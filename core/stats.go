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
	"math"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"rcproxy/core/codec"
)

var GlobalStats ProxyStats

type ConnCloseType int

const (
	// the client actively closes the connection
	ConnEof ConnCloseType = iota
	// proxy and client connection error
	ConnErr
	// proxy actively closes the connection
	ProxyEof
)

type ProxyStats struct {
	Request *prometheus.HistogramVec

	TotalConnections *prometheus.CounterVec
	CurrConnections  *prometheus.GaugeVec
	TotalRequests    *prometheus.CounterVec

	ClientConnectionsClientEof *prometheus.CounterVec
	ClientConnectionsClientErr *prometheus.CounterVec

	ServerEjects *prometheus.CounterVec
	ForwardErr   *prometheus.CounterVec
	Fragments    *prometheus.CounterVec

	ReqCmd *prometheus.CounterVec

	RedisServerEof             *prometheus.CounterVec
	RedisServerErr             *prometheus.CounterVec
	RedisServerActive          *prometheus.GaugeVec
	RedisServerCreateConnError *prometheus.CounterVec

	TimeoutTree *prometheus.GaugeVec
}

func init() {
	GlobalStats = NewProxyStats("rcproxy")
}

func NewProxyStats(namespace string) ProxyStats {
	stats := ProxyStats{
		TotalConnections: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "total_connections",
			Help:      "total connections",
		}, nil),
		CurrConnections: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "curr_connections",
			Help:      "current connections",
		}, []string{"type"}),
		TotalRequests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "total_requests",
			Help:      "total requests",
		}, nil),
		Request: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "request_latency",
			Help:      "request latency",
			Buckets:   []float64{10, 20, 50, 100, 200, 500},
		}, nil),
		ClientConnectionsClientEof: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "client_connections_client_eof",
			Help:      "client actively closes the connection",
		}, nil),
		ClientConnectionsClientErr: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "client_connections_client_err",
			Help:      "client connection error",
		}, nil),
		ReqCmd: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "cmd",
			Help:      "number of redis command requests",
		}, []string{"cmd"}),
		Fragments: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "fragments",
			Help:      "fragments created from a multi-vector request",
		}, []string{"cmd"}),
		RedisServerEof: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "redis_connections_eof",
			Help:      "redis actively closes the connection to the proxy",
		}, []string{"addr"}),
		RedisServerErr: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "redis_connections_err",
			Help:      "redis connection error",
		}, []string{"addr"}),
		RedisServerCreateConnError: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "redis_connections_create_conn_error",
			Help:      "number of connection timeouts between proxy and redis",
		}, []string{"addr"}),
		RedisServerActive: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "redis_connections_active",
			Help:      "number of active connections between proxy and redis",
		}, []string{"addr"}),
		TimeoutTree: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "timeout_tree",
			Help:      "timeout tree health level",
		}, []string{"type"}),
	}
	prometheus.MustRegister(
		stats.TotalConnections, stats.CurrConnections, stats.TotalRequests,
		stats.ClientConnectionsClientEof, stats.ClientConnectionsClientErr,
		stats.RedisServerCreateConnError, stats.RedisServerEof, stats.RedisServerErr,
		stats.RedisServerActive, stats.Request, stats.TimeoutTree, stats.ReqCmd,
	)
	return stats
}

func (s *ProxyStats) ReqCmdIncr(cmd codec.Command) {
	switch cmd {
	// for del
	case codec.ReqDel:
		GlobalStats.ReqCmd.WithLabelValues(codec.Transform2Str(cmd)).Inc()
	// for uniq key
	case codec.ReqGet, codec.ReqSet, codec.ReqMget, codec.ReqMset, codec.ReqSort:
		GlobalStats.ReqCmd.WithLabelValues(codec.Transform2Str(cmd)).Inc()
		fallthrough
	// for string
	case codec.ReqSetex, codec.ReqSetnx, codec.ReqSetrange, codec.ReqGetrange, codec.ReqStrlen:
		GlobalStats.ReqCmd.WithLabelValues("string").Inc()

	// for bitmap
	case codec.ReqBitcount, codec.ReqSetbit, codec.ReqGetbit:
		GlobalStats.ReqCmd.WithLabelValues("bitmap").Inc()

	// for incr/decr
	case codec.ReqIncr, codec.ReqDecr, codec.ReqDecrby, codec.ReqIncrby, codec.ReqIncrbyfloat:
		GlobalStats.ReqCmd.WithLabelValues("incr_decr").Inc()

	// for hash
	case codec.ReqHexists, codec.ReqHget, codec.ReqHgetall, codec.ReqHkeys, codec.ReqHlen, codec.ReqHmget, codec.ReqHmset, codec.ReqHdel:
		fallthrough
	case codec.ReqHincrby, codec.ReqHincrbyfloat, codec.ReqHset, codec.ReqHsetnx, codec.ReqHscan, codec.ReqHvals:
		GlobalStats.ReqCmd.WithLabelValues("hashs").Inc()

	// for list
	case codec.ReqLrem:
		GlobalStats.ReqCmd.WithLabelValues(codec.Transform2Str(cmd)).Inc()
		fallthrough
	case codec.ReqLpush, codec.ReqRpush, codec.ReqRpushx, codec.ReqLpushx, codec.ReqLpop, codec.ReqRpop, codec.ReqRpoplpush:
		fallthrough
	case codec.ReqLrange, codec.ReqLset, codec.ReqLtrim, codec.ReqLindex, codec.ReqLlen, codec.ReqLinsert:
		GlobalStats.ReqCmd.WithLabelValues("lists").Inc()

	// for set
	case codec.ReqSadd, codec.ReqSpop, codec.ReqSrem, codec.ReqSscan, codec.ReqSmove:
		fallthrough
	case codec.ReqSrandmember, codec.ReqScard, codec.ReqSismember, codec.ReqSmembers:
		fallthrough
	case codec.ReqSunion, codec.ReqSdiff, codec.ReqSinter, codec.ReqSinterstore, codec.ReqSdiffstore, codec.ReqSunionstore:
		GlobalStats.ReqCmd.WithLabelValues("sets").Inc()

	// for zset
	case codec.ReqZadd, codec.ReqZcount, codec.ReqZincrby, codec.ReqZscan, codec.ReqZcard, codec.ReqZscore:
		fallthrough
	case codec.ReqZrange, codec.ReqZrank, codec.ReqZrangebyscore, codec.ReqZrevrange, codec.ReqZrangebylex, codec.ReqZrevrank:
		fallthrough
	case codec.ReqZinterstore, codec.ReqZrevrangebyscore, codec.ReqZunionstore, codec.ReqZremrangebyscore:
		fallthrough
	case codec.ReqZrem, codec.ReqZremrangebylex, codec.ReqZremrangebyrank:
		GlobalStats.ReqCmd.WithLabelValues("sortedsets").Inc()

	default:
		GlobalStats.ReqCmd.WithLabelValues("other").Inc()
	}
}

// statsLoop some statistics do not need to be put into the event loop, split out and executed per second
func statsLoop() {
	ticker := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-ticker.C:
			depth, stddev := depthOfTimeoutQueue()
			GlobalStats.TimeoutTree.WithLabelValues("length").Set(lengthOfTimeoutQueue())
			if math.IsNaN(depth) {
				depth = 0
			}
			if math.IsNaN(stddev) {
				stddev = 0
			}
			GlobalStats.TimeoutTree.WithLabelValues("depth").Set(depth)
			GlobalStats.TimeoutTree.WithLabelValues("stddev").Set(stddev)

			cConnCount := float64(EngineGlobal.eng.el.loadCConn())
			sConnCount := float64(EngineGlobal.eng.el.loadSConn())
			GlobalStats.CurrConnections.WithLabelValues("client").Set(cConnCount)
			GlobalStats.CurrConnections.WithLabelValues("server").Set(sConnCount)
			GlobalStats.CurrConnections.WithLabelValues("total").Set(cConnCount + sConnCount)
		}
	}
}
