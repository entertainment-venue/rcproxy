# redis-test set DSL

DEL myset "{myset}1" "{myset}2"

SCARD myset
RET 0

SADD myset "hello"
RET 1

SADD myset "foo"
RET 1

SADD myset "hello"
RET 0

SCARD myset
RET 2

SPOP myset
SCARD myset
RET 1

SADD "{myset}1" "hello"
SADD "{myset}1" "foo"

SADD "{myset}2" "hello"
SADD "{myset}2" "bar"

SDIFF "{myset}1" "{myset}2"
RET [foo]

SINTER "{myset}1" "{myset}2"
RET [hello]

SUNION "{myset}1" "{myset}2"
RET_LEN 3

SREM "{myset}1" hello
RET 1

SSCAN "{myset}1" 0
RET [0, [foo]]
