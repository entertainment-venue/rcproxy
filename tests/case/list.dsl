# redis-test list DSL

DEL mylist
LPUSH mylist "World"
RET 1
LPUSH mylist "Hello"
RET 2
LINDEX mylist 0
RET Hello
LINDEX mylist -1
RET World
LINDEX mylist 3
RET nil
DEL mylist
RET 1

RPUSH mylist "Hello"
RPUSH mylist "World"
LINSERT mylist BEFORE "World" "There"
RET 3
LLEN mylist
RET 3
LRANGE mylist 0 -1
RET [Hello There World]
LPOP mylist
RET Hello
LLEN mylist
RET 2
DEL mylist
RET 1

RPUSH mylist "hello"
RPUSH mylist "hello"
RPUSH mylist "foo"
RPUSH mylist "hello"
RET 4
LREM mylist -2 "hello"
RET 2
LRANGE mylist 0 -1
RET [hello, foo]
LSET mylist 0 "bar"
RET OK
LRANGE mylist 0 -1
RET [bar, foo]
DEL mylist
RET 1

RPUSH mylist "hello"
RPUSH mylist "hello"
RPUSH mylist "foo"
RPUSH mylist "bar"
RET 4
LTRIM mylist 1 -1
RET OK
LRANGE mylist 0 -1
RET [hello, foo, bar]
DEL mylist
RET 1

RPUSH mylist "hello"
RPUSH mylist "foo"
RET 2
RPUSHX mylist2 "bar"
RET 0
LRANGE mylist 0 -1
RET [hello, foo]
DEL mylist
RET 1
