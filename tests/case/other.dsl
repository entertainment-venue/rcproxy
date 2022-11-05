# redis-test other DSL

PING
RET PONG

SET Foo Bar
RET OK

EXISTS Foo
RET 1

TYPE Foo
RET string

DEL Foo
RET 1

EXISTS Foo
RET 0

AUTH passwd123
RET OK

AUTH 123456
RET "ERR invalid password"

QUIT
RET OK