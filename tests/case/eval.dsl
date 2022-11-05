# redis-test eval DSL

SET a 1
RET OK
SET qnzy 2
RET OK
SET smhp 3
RET OK

EVAL "return {redis.pcall('get', 'a'), redis.pcall('set', 'qnzy', 'c'), redis.pcall('get', 'smhp')}" 1 a
RET ["1", "OK", "3"]

GET qnzy
RET c

DEL a qnzy smhp
RET 3