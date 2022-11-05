## Redis Command Support

### Note
- redis commands are not case sensitive
- only vectored commands 'MGET key [key ...]', 'MSET key value [key value ...]', 'DEL key [key ...]' needs to be fragmented.

### Keys Command

| Command    | Supported? |  Comment  |
| :--------: | :--------: |  :----   |
| DEL | Yes | |
| DUMP | Yes | |
| EXISTS | Yes | EXISTS key |
| EXISTS | No | EXISTS key [key ...] |
| EXPIRE | Yes | |
| EXPIREAT | Yes | |
| KEYS | No | |
| MIGRATE | No | |
| MOVE | No | |
| OBJECT | No | |
| PERSIST | Yes | |
| PEXPIRE | Yes | |
| PEXPIREAT | Yes | |
| PTTL | Yes | |
| RANDOMKEY | No | |
| RENAME | No | |
| RENAMENX | No | |
| RESTORE | Yes | |
| SORT | Yes | |
| TTL | Yes | |
| TYPE | Yes | |
| SCAN | No | |

### Strings Command

| Command    | Supported? |  Comment  |
| :--------: | :--------: |  :----   |
| APPEND | Yes | |
| BITCOUNT | Yes | |
| BITFIELD | No | |
| BITOP | No | |
| BITPOS | No | |
| DECR | Yes | |
| DECRBY | Yes | |
| GET | Yes | |
| GETBIT | Yes | |
| GETDEL | No | |
| GETEX | No | |
| GETRANGE | Yes | |
| GETSET | Yes | |
| INCR | Yes | |
| INCRBY | Yes | |
| INCRBYFLOAT | Yes | |
| MGET | Yes | |
| MSET | Yes | |
| MSETNX | No | |
| PSETEX | Yes | |
| SET | Yes | |
| SETBIT | Yes | |
| SETEX | Yes | |
| SETNX | Yes | |
| SETRANGE | Yes | |
| STRALGO | No | |
| STRLEN | Yes | |

> MSET support is not Atomic

### Hashes Command

| Command    | Supported? |  Comment  |
| :--------: | :--------: |  :----   |
| HDEL | Yes | |
| HEXISTS | Yes | |
| HGET | Yes | |
| HGETALL | Yes | |
| HINCRBY | Yes | |
| HINCRBYFLOAT | Yes | |
| HKEYS | Yes | |
| HLEN | Yes | |
| HMGET | Yes | |
| HMSET | Yes | |
| HSET | Yes | |
| HSETNX | Yes | |
| HVALS | Yes | |
| HSCAN | Yes | |

### Lists Command

| Command    | Supported? |  Comment  |
| :--------: | :--------: |  :----   |
| BLMOVE | No | |
| BLPOP | No | |
| BRPOP | No | |
| BRPOPLPUSH | No | |
| LINDEX | Yes | |
| LINSERT | Yes | |
| LLEN | Yes | |
| LMOVE | No | |
| LPOP | Yes | |
| LPOS | No | |
| LPUSH | Yes | |
| LPUSHX | Yes | |
| LRANGE | Yes | |
| LREM | Yes | |
| LSET | Yes | |
| LTRIM | Yes | |
| RPOP | Yes | |
| RPOPLPUSH | Yes | |
| RPUSH | Yes | |
| RPUSHX | Yes | |

### Sets Command

| Command    | Supported? |  Comment  |
| :--------: | :--------: |  :----   |
| SADD | Yes | |
| SCARD | Yes | |
| SDIFF | Yes | |
| SDIFFSTORE | Yes | |
| SINTER | Yes | |
| SINTERSTORE | Yes | |
| SISMEMBER | Yes | |
| SMEMBERS | Yes | |
| SMISMEMBER | No | |
| SMOVE | Yes | |
| SPOP | Yes | |
| SRANDMEMBER | Yes | |
| SREM | Yes | |
| SSCAN | Yes | |
| SUNION | Yes | |
| SUNIONSTORE | Yes | |

### Sorted Sets Command

| Command    | Supported? |  Comment  |
| :--------: | :--------: |  :----   |
| BZPOPMAX | No | |
| BZPOPMIN | No | |
| ZADD | Yes | |
| ZCARD | Yes | |
| ZCOUNT | Yes | |
| ZDIFF | No | |
| ZDIFFSTORE | No | |
| ZINCRBY | Yes | |
| ZINTER | No | |
| ZINTERSTORE | Yes| |
| ZLEXCOUNT | Yes | |
| ZMSCORE | No | |
| ZPOPMAX | No | |
| ZPOPMIN | No | |
| ZRANDMEMBER | No | |
| ZRANGE | Yes | |
| ZRANGEBYLEX | Yes | |
| ZRANGEBYSCORE | Yes | |
| ZRANGESTORE | No | |
| ZRANK | Yes | |
| ZREM | Yes | |
| ZREMRANGEBYLEX | Yes | |
| ZREMRANGEBYRANK | Yes | |
| ZREMRANGEBYSCORE | Yes | |
| ZREVRANGE | Yes | |
| ZREVRANGEBYLEX | No | |
| ZREVRANGEBYSCORE | Yes | |
| ZREVRANK | Yes | |
| ZSCAN | Yes | |
| ZSCORE | Yes | |
| ZUNION | No | |
| ZUNIONSTORE | Yes | |

### HyperLogLog Command

| Command    | Supported? |  Comment  |
| :--------: | :--------: |  :----   |
| PFADD | Yes | |
| PFCOUNT | Yes | |
| PFMERGE | Yes | |

### Geo Command

| Command    | Supported? |  Comment  |
| :--------: | :--------: |  :----   |
| GEOADD | No | |
| GEODIST | No | |
| GEOHASH | No | |
| GEOPOS | No | |
| GEORADIUS | No | |
| GEORADIUSBYMEMBER | No | |
| GEOSEARCH | No | |
| GEOSEARCHSTORE | No | |

### Pub/Sub Command

| Command    | Supported? |  Comment  |
| :--------: | :--------: |  :----   |
| PSUBSCRIBE | No | |
| PUBLISH | No | |
| PUBSUB | No | |
| PUNSUBSCRIBE | No | |
| SUBSCRIBE | No | |
| UNSUBSCRIBE | No | |

### Transactions Command

| Command    | Supported? |  Comment  |
| :--------: | :--------: |  :----   |
| DISCARD | No | |
| EXEC | No | |
| MULTI | No | |
| UNWATCH | No | |
| WATCH | No | |

### Scripting Command

| Command    | Supported? |  Comment  |
| :--------: | :--------: |  :----   |
| EVAL | Yes | |
| EVALSHA | Yes | |
| SCRIPT DEBUG | No | |
| SCRIPT EXISTS | No | |
| SCRIPT FLUSH | No | |
| SCRIPT KILL | No | |
| SCRIPT LOAD | No | |

### Connection Command

| Command    | Supported? |  Comment  |
| :--------: | :--------: |  :----   |
| AUTH | No | |
| ECHO | No | |
| PING | Yes | |
| QUIT | Yes | |
| SELECT | No | |

### Server Command

| Command    | Supported? |  Comment  |
| :--------: | :--------: |  :----   |
| BGREWRITEAOF | No | |
| BGSAVE | No | |
| CLIENT KILL | No | |
| CLIENT LIST | No | |
| CONFIG GET | No | |
| CONFIG SET | No | |
| CONFIG RESETSTAT | No | |
| DBSIZE | No | |
| DEBUG OBJECT | No | |
| DEBUG SEGFAULT | No | |
| FLUSHALL | No | |
| FLUSHDB | No | |
| INFO | No | |
| LASTSAVE | No | |
| MONITOR | No | |
| SAVE | No | |
| SHUTDOWN | No | |
| SLAVEOF | No | |
| SLOWLOG | No | |
| SYNC | No | |
| TIME | No | |
| COMMAND | No | |
| LOLWUT | No | |