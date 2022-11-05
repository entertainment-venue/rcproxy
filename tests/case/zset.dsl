# redis-test set DSL

DEL myzset myzset2

ZCARD myzset
RET 0

ZADD myzset 1 "one"
RET 1

ZADD myzset 1 "uno"
RET 1

ZADD myzset 2 "two" 3 "three"
RET 2

ZRANGE myzset 0 -1 WITHSCORES
RET [one, 1, uno, 1, two, 2, three, 3]

ZCARD myzset
RET 4

ZCOUNT myzset 1 2
RET 3

ZINCRBY myzset 3 "one"
RET 4

ZRANGE myzset 0 -1 WITHSCORES
RET [uno, 1, two, 2, three, 3, one, 4]

ZADD myzset2 0 a 0 b 0 c 0 d 0 e
RET 5

ZADD myzset2 0 f 0 g
RET 2

ZLEXCOUNT myzset2 "-" "+"
RET 7

ZLEXCOUNT myzset2 "[b" "[f"
RET 5

ZRANGEBYLEX myzset2 "-" "[c"
RET [a, b, c]

ZRANGEBYLEX myzset2 "-" "(c"
RET [a, b]

ZRANGEBYLEX myzset2 "[aaa" "(g"
RET [b, c, d, e, f]

DEL salary

ZADD salary 2500 jack
RET 1
ZADD salary 5000 tom
RET 1
ZADD salary 12000 peter
RET 1

ZRANGEBYSCORE salary "-inf" "+inf"
RET [jack, tom, peter]

ZRANGEBYSCORE salary "-inf" "+inf" WITHSCORES
RET [jack, 2500, tom, 5000, peter, 12000]

ZRANGEBYSCORE salary "-inf" 5000 WITHSCORES
RET [jack, 2500, tom, 5000]

ZRANGEBYSCORE salary "(5000" 400000
RET [peter]

ZRANGE salary 0 -1 WITHSCORES
RET [jack, 2500, tom, 5000, peter, 12000]

ZRANK salary tom
RET 1

ZCARD salary
RET 3

ZREM salary tom peter
RET 2

ZCARD salary
RET 1

DEL salary
RET 1

DEL myzset myzset2
RET 2
