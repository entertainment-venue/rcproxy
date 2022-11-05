// Copyright (c) 2022 The rcproxy Authors
// Copyright (c) 2011 Twitter, Inc.
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

package codec

type Command uint32
type NArgs int

const (
	UNKNOWN   Command = iota
	ReqExists         /* redis commands - keys */
	ReqTtl
	ReqPttl
	ReqType
	ReqDump
	ReqBitcount /* redis requests - string */
	ReqGet
	ReqGetbit
	ReqGetrange
	ReqMget
	ReqStrlen
	ReqHexists /* redis requests - hash */
	ReqHget
	ReqHgetall
	ReqHkeys
	ReqHlen
	ReqHmget
	ReqHscan
	ReqHvals
	ReqLindex /* redis requests - lists */
	ReqLlen
	ReqLrange
	ReqSrandmember /* redis requests - set */
	ReqSscan
	ReqSdiff
	ReqSinter
	ReqScard
	ReqSismember
	ReqSmembers
	ReqZcard /* redis requests - sorted set */
	ReqZcount
	ReqZlexcount
	ReqZrange
	ReqZrangebylex
	ReqZrangebyscore
	ReqZrank
	ReqZrevrange
	ReqZrevrangebyscore
	ReqZrevrank
	ReqZscore
	ReqZscan

	ReqWriteCmdStart /* redis write commands below */
	ReqDel           /* redis commands - keys */
	ReqExpire
	ReqExpireat
	ReqPexpire
	ReqPexpireat
	ReqPersist
	ReqSort
	ReqAppend /* redis requests - string */
	ReqDecr
	ReqDecrby
	ReqGetset
	ReqIncr
	ReqIncrby
	ReqIncrbyfloat
	ReqMset
	ReqPsetex
	ReqRestore
	ReqSet
	ReqSetbit
	ReqSetex
	ReqSetnx
	ReqSetrange
	ReqSunion
	ReqHdel /* redis requests - hashes */
	ReqHincrby
	ReqHincrbyfloat
	ReqHmset
	ReqHset
	ReqHsetnx
	ReqLinsert /* redis requests - list */
	ReqLpop
	ReqLpush
	ReqLpushx
	ReqLrem
	ReqLset
	ReqLtrim
	ReqRpop
	ReqRpoplpush
	ReqRpush
	ReqRpushx
	ReqPfadd /* redis requests - hyperloglog */
	ReqPfcount
	ReqPfmerge
	ReqSadd /* redis requests - sets */
	ReqSdiffstore
	ReqSinterstore
	ReqSmove
	ReqSpop
	ReqSrem
	ReqSunionstore
	ReqZadd /* redis requests - sorted sets */
	ReqZincrby
	ReqZinterstore
	ReqZrem
	ReqZremrangebyrank
	ReqZremrangebylex
	ReqZremrangebyscore
	ReqZunionstore
	ReqEval /* redis requests - eval */
	ReqEvalsha
	ReqPing /* redis requests - ping/quit */
	ReqQuit
	ReqAuth
	ReqTooLarge
	ReqWrongArgumentsNumber

	RspTooLarge
	RspStatus /* redis response */
	RspOk
	RspPong
	RspError
	RspNeedAuth
	RspNeedNtAuth // needn't auth
	RspAuthFailed
	RspInteger
	RspBulk
	RspMultibulk
	RspAsk
	RspMoved
	Sentinel
)

const (
	Nargsz       NArgs = 0  // 0 key, 0 parameter
	Nargs0       NArgs = 1  // 1 key, 0 parameter
	Nargs1       NArgs = 2  // 1 key, 1 parameter
	Nargs2       NArgs = 3  // 1 key, 2 parameter
	Nargs3       NArgs = 4  // 1 key, 3 parameter
	NargsInf     NArgs = -1 // 1 key, unlimited parameter
	NargsEvenInf NArgs = -2 // 1 key, unlimited even parameter
)

var CommandType2Str = map[Command]string{
	ReqExists:           "exists",
	ReqTtl:              "ttl",
	ReqPttl:             "pttl",
	ReqType:             "type",
	ReqDump:             "dump",
	ReqBitcount:         "bitcount",
	ReqGet:              "get",
	ReqGetbit:           "getbit",
	ReqGetrange:         "getrange",
	ReqMget:             "mget",
	ReqStrlen:           "strlen",
	ReqHexists:          "hexists",
	ReqHget:             "hget",
	ReqHgetall:          "hgetall",
	ReqHkeys:            "hkeys",
	ReqHlen:             "hlen",
	ReqHmget:            "hmget",
	ReqHscan:            "hscan",
	ReqHvals:            "hvals",
	ReqLindex:           "lindex",
	ReqLlen:             "llen",
	ReqLrange:           "lrange",
	ReqSrandmember:      "srandmember",
	ReqSscan:            "sscan",
	ReqSdiff:            "sdiff",
	ReqSinter:           "sinter",
	ReqScard:            "scard",
	ReqSismember:        "sismember",
	ReqSmembers:         "smembers",
	ReqZcard:            "zcard",
	ReqZcount:           "zcount",
	ReqZlexcount:        "zlexcount",
	ReqZrange:           "zrange",
	ReqZrangebylex:      "zrangebylex",
	ReqZrangebyscore:    "zrangebyscore",
	ReqZrank:            "zrank",
	ReqZrevrange:        "zrevrange",
	ReqZrevrangebyscore: "zrevrangebyscore",
	ReqZrevrank:         "zrevrank",
	ReqZscore:           "zscore",
	ReqZscan:            "zscan",

	ReqDel:              "del",
	ReqExpire:           "expire",
	ReqExpireat:         "expireat",
	ReqPexpire:          "pexpire",
	ReqPexpireat:        "pexpireat",
	ReqPersist:          "persist",
	ReqSort:             "sort",
	ReqAppend:           "append",
	ReqDecr:             "decr",
	ReqDecrby:           "decrby",
	ReqGetset:           "getset",
	ReqIncr:             "incr",
	ReqIncrby:           "incrby",
	ReqIncrbyfloat:      "incrbyfloat",
	ReqMset:             "mset",
	ReqPsetex:           "psetex",
	ReqRestore:          "restore",
	ReqSet:              "set",
	ReqSetbit:           "setbit",
	ReqSetex:            "setex",
	ReqSetnx:            "setnx",
	ReqSetrange:         "setrange",
	ReqSunion:           "sunion",
	ReqHdel:             "hdel",
	ReqHincrby:          "hincrby",
	ReqHincrbyfloat:     "hincrbyfloat",
	ReqHmset:            "hmset",
	ReqHset:             "hset",
	ReqHsetnx:           "hsetnx",
	ReqLinsert:          "linsert",
	ReqLpop:             "lpop",
	ReqLpush:            "lpush",
	ReqLpushx:           "lpushx",
	ReqLrem:             "lrem",
	ReqLset:             "lset",
	ReqLtrim:            "ltrim",
	ReqRpop:             "rpop",
	ReqRpoplpush:        "rpoplpush",
	ReqRpush:            "rpush",
	ReqRpushx:           "rpushx",
	ReqPfadd:            "pfadd",
	ReqPfcount:          "pfcount",
	ReqPfmerge:          "pfmerge",
	ReqSadd:             "sadd",
	ReqSdiffstore:       "sdiffstore",
	ReqSinterstore:      "sinterstore",
	ReqSmove:            "smove",
	ReqSpop:             "spop",
	ReqSrem:             "srem",
	ReqSunionstore:      "sunionstore",
	ReqZadd:             "zadd",
	ReqZincrby:          "zincrby",
	ReqZinterstore:      "zinterstore",
	ReqZrem:             "zrem",
	ReqZremrangebyrank:  "zremrangebyrank",
	ReqZremrangebylex:   "zremrangebylex",
	ReqZremrangebyscore: "zremrangebyscore",
	ReqZunionstore:      "zunionstore",
	ReqEval:             "eval",
	ReqEvalsha:          "evalsha",
	ReqPing:             "ping",
	ReqQuit:             "quit",
	ReqAuth:             "auth",
}

var CommandStr2Type = map[string]Command{
	"exists":           ReqExists,
	"ttl":              ReqTtl,
	"pttl":             ReqPttl,
	"type":             ReqType,
	"dump":             ReqDump,
	"bitcount":         ReqBitcount,
	"get":              ReqGet,
	"getbit":           ReqGetbit,
	"getrange":         ReqGetrange,
	"mget":             ReqMget,
	"strlen":           ReqStrlen,
	"hexists":          ReqHexists,
	"hget":             ReqHget,
	"hgetall":          ReqHgetall,
	"hkeys":            ReqHkeys,
	"hlen":             ReqHlen,
	"hmget":            ReqHmget,
	"hscan":            ReqHscan,
	"hvals":            ReqHvals,
	"lindex":           ReqLindex,
	"llen":             ReqLlen,
	"lrange":           ReqLrange,
	"srandmember":      ReqSrandmember,
	"sscan":            ReqSscan,
	"sdiff":            ReqSdiff,
	"sinter":           ReqSinter,
	"scard":            ReqScard,
	"sismember":        ReqSismember,
	"smembers":         ReqSmembers,
	"zcard":            ReqZcard,
	"zcount":           ReqZcount,
	"zlexcount":        ReqZlexcount,
	"zrange":           ReqZrange,
	"zrangebylex":      ReqZrangebylex,
	"zrangebyscore":    ReqZrangebyscore,
	"zrank":            ReqZrank,
	"zrevrange":        ReqZrevrange,
	"zrevrangebyscore": ReqZrevrangebyscore,
	"zrevrank":         ReqZrevrank,
	"zscore":           ReqZscore,
	"zscan":            ReqZscan,

	"del":              ReqDel,
	"expire":           ReqExpire,
	"expireat":         ReqExpireat,
	"pexpire":          ReqPexpire,
	"pexpireat":        ReqPexpireat,
	"persist":          ReqPersist,
	"sort":             ReqSort,
	"append":           ReqAppend,
	"decr":             ReqDecr,
	"decrby":           ReqDecrby,
	"getset":           ReqGetset,
	"incr":             ReqIncr,
	"incrby":           ReqIncrby,
	"incrbyfloat":      ReqIncrbyfloat,
	"mset":             ReqMset,
	"psetex":           ReqPsetex,
	"restore":          ReqRestore,
	"set":              ReqSet,
	"setbit":           ReqSetbit,
	"setex":            ReqSetex,
	"setnx":            ReqSetnx,
	"setrange":         ReqSetrange,
	"sunion":           ReqSunion,
	"hdel":             ReqHdel,
	"hincrby":          ReqHincrby,
	"hincrbyfloat":     ReqHincrbyfloat,
	"hmset":            ReqHmset,
	"hset":             ReqHset,
	"hsetnx":           ReqHsetnx,
	"linsert":          ReqLinsert,
	"lpop":             ReqLpop,
	"lpush":            ReqLpush,
	"lpushx":           ReqLpushx,
	"lrem":             ReqLrem,
	"lset":             ReqLset,
	"ltrim":            ReqLtrim,
	"rpop":             ReqRpop,
	"rpoplpush":        ReqRpoplpush,
	"rpush":            ReqRpush,
	"rpushx":           ReqRpushx,
	"pfadd":            ReqPfadd,
	"pfcount":          ReqPfcount,
	"pfmerge":          ReqPfmerge,
	"sadd":             ReqSadd,
	"sdiffstore":       ReqSdiffstore,
	"sinterstore":      ReqSinterstore,
	"smove":            ReqSmove,
	"spop":             ReqSpop,
	"srem":             ReqSrem,
	"sunionstore":      ReqSunionstore,
	"zadd":             ReqZadd,
	"zincrby":          ReqZincrby,
	"zinterstore":      ReqZinterstore,
	"zrem":             ReqZrem,
	"zremrangebyrank":  ReqZremrangebyrank,
	"zremrangebylex":   ReqZremrangebylex,
	"zremrangebyscore": ReqZremrangebyscore,
	"zunionstore":      ReqZunionstore,
	"eval":             ReqEval,
	"evalsha":          ReqEvalsha,
	"ping":             ReqPing,
	"quit":             ReqQuit,
	"auth":             ReqAuth,
}

var CommandType2ArgsNumber = map[Command]NArgs{
	ReqPing: Nargsz,
	ReqQuit: Nargsz,

	ReqExists:   Nargs0,
	ReqTtl:      Nargs0,
	ReqPttl:     Nargs0,
	ReqType:     Nargs0,
	ReqDump:     Nargs0,
	ReqGet:      Nargs0,
	ReqStrlen:   Nargs0,
	ReqHgetall:  Nargs0,
	ReqHkeys:    Nargs0,
	ReqHlen:     Nargs0,
	ReqSmembers: Nargs0,
	ReqZcard:    Nargs0,
	ReqLlen:     Nargs0,
	ReqScard:    Nargs0,
	ReqHvals:    Nargs0,
	ReqPfcount:  Nargs0,
	ReqSpop:     Nargs0,
	ReqAuth:     Nargs0,
	ReqRpop:     Nargs0,
	ReqPersist:  Nargs0,
	ReqDecr:     Nargs0,
	ReqIncr:     Nargs0,
	ReqLpop:     Nargs0,

	ReqRpoplpush:   Nargs1,
	ReqRpushx:      Nargs1,
	ReqGetbit:      Nargs1,
	ReqHexists:     Nargs1,
	ReqHget:        Nargs1,
	ReqLindex:      Nargs1,
	ReqSismember:   Nargs1,
	ReqExpire:      Nargs1,
	ReqZrank:       Nargs1,
	ReqZrevrank:    Nargs1,
	ReqZscore:      Nargs1,
	ReqExpireat:    Nargs1,
	ReqPexpire:     Nargs1,
	ReqPexpireat:   Nargs1,
	ReqAppend:      Nargs1,
	ReqDecrby:      Nargs1,
	ReqGetset:      Nargs1,
	ReqIncrby:      Nargs1,
	ReqIncrbyfloat: Nargs1,
	ReqSetnx:       Nargs1,
	ReqLpushx:      Nargs1,

	ReqGetrange:         Nargs2,
	ReqLrange:           Nargs2,
	ReqZcount:           Nargs2,
	ReqZlexcount:        Nargs2,
	ReqPsetex:           Nargs2,
	ReqRestore:          Nargs2,
	ReqSetbit:           Nargs2,
	ReqSetex:            Nargs2,
	ReqSetrange:         Nargs2,
	ReqHincrby:          Nargs2,
	ReqHincrbyfloat:     Nargs2,
	ReqHset:             Nargs2,
	ReqHsetnx:           Nargs2,
	ReqLrem:             Nargs2,
	ReqLset:             Nargs2,
	ReqLtrim:            Nargs2,
	ReqSmove:            Nargs2,
	ReqZincrby:          Nargs2,
	ReqZremrangebyrank:  Nargs2,
	ReqZremrangebylex:   Nargs2,
	ReqZremrangebyscore: Nargs2,

	ReqLinsert: Nargs3,

	ReqSet:              NargsInf,
	ReqHmset:            NargsInf,
	ReqLpush:            NargsInf,
	ReqSunion:           NargsInf,
	ReqHdel:             NargsInf,
	ReqPfmerge:          NargsInf,
	ReqRpush:            NargsInf,
	ReqPfadd:            NargsInf,
	ReqSadd:             NargsInf,
	ReqSdiffstore:       NargsInf,
	ReqSinterstore:      NargsInf,
	ReqSrem:             NargsInf,
	ReqSunionstore:      NargsInf,
	ReqZadd:             NargsInf,
	ReqZinterstore:      NargsInf,
	ReqZrem:             NargsInf,
	ReqBitcount:         NargsInf,
	ReqZunionstore:      NargsInf,
	ReqEval:             NargsInf,
	ReqEvalsha:          NargsInf,
	ReqMget:             NargsInf,
	ReqHmget:            NargsInf,
	ReqHscan:            NargsInf,
	ReqSrandmember:      NargsInf,
	ReqSscan:            NargsInf,
	ReqSdiff:            NargsInf,
	ReqSinter:           NargsInf,
	ReqZrange:           NargsInf,
	ReqZrangebylex:      NargsInf,
	ReqZrangebyscore:    NargsInf,
	ReqZrevrange:        NargsInf,
	ReqZrevrangebyscore: NargsInf,
	ReqZscan:            NargsInf,
	ReqDel:              NargsInf,
	ReqSort:             NargsInf,

	ReqMset: NargsEvenInf,
}

func Transform2Type(command []byte, n int) Command {
	toLower(command)
	if v, ok := CommandStr2Type[string(command)]; ok {
		return checkArgs(v, n)
	}
	return UNKNOWN
}

func Transform2Str(command Command) string {
	if v, ok := CommandType2Str[command]; ok {
		return v
	}
	return "unknown"
}

func checkArgs(command Command, n int) Command {
	nargs, ok := CommandType2ArgsNumber[command]
	if !ok {
		return ReqWrongArgumentsNumber
	}

	switch nargs {
	case Nargsz, Nargs0, Nargs1, Nargs2, Nargs3:
		if int(nargs) != n {
			return ReqWrongArgumentsNumber
		}
	case NargsInf:
		if n < 1 {
			return ReqWrongArgumentsNumber
		}
	case NargsEvenInf:
		if n < 2 || n%2 == 1 {
			return ReqWrongArgumentsNumber
		}
	default:
		return ReqWrongArgumentsNumber
	}
	return command
}

// toLower the method is faster than strings.ToLower because it eliminates one copy
func toLower(bs []byte) {
	for i := 0; i < len(bs); i++ {
		if bs[i] >= 'A' && bs[i] <= 'Z' {
			bs[i] = bs[i] ^ 0x20
		}
	}
}
