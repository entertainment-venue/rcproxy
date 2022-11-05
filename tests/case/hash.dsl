# redis-test hash DSL

DEL Foo

HMSET Foo name "redis tutorial" description "for caching" likes 20 visitors 13
RET OK

HGETALL Foo
RET ["name", "redis tutorial", "description", "for caching", "likes" , "20", "visitors", "13"]

HEXISTS Foo description
RET 1

HDEL Foo description likes
RET 2

HEXISTS Foo description
RET 0

HGETALL Foo
RET ["name", "redis tutorial", "visitors", "13"]

HKEYS Foo
RET ["name", "visitors"]

HLEN Foo
RET 2

HMGET Foo visitors
RET ["13"]

HINCRBY Foo visitors 2
RET 15

HSETNX Foo likes 10
RET 1

HSETNX Foo likes 11
RET 0

HVALS Foo
RET ["redis tutorial", "15", "10"]

DEL Foo
RET 1