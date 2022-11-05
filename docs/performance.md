## Performance

### Setup

+ redis-cluster-server (version 3.0.3) running on machine A.
+ nutcracker running on machine A as a local proxy to redis-cluster-server.
+ rcproxy running on machine A as a local proxy to redis-cluster-server.
+ redis-benchmark running on machine B.
+ machine A != machine B.
+ nutcracker built with --enable-debug=no
+ nutcracker running with mbuf-size of 512 (-m 512)
+ rcproxy running with LOG_LEVEL=INFO.

### redis-benchmark against redis-server

    $ redis-benchmark -h <machine-A> -q -t set,get,incr,lpush,lpop,sadd,spop,lpush,lrange -r 16384 -c 100 -p 6379
    SET: 72674.41 requests per second
    GET: 70028.02 requests per second
    INCR: 72516.32 requests per second
    LPUSH: 72939.46 requests per second
    LPOP: 73964.50 requests per second
    SADD: 71942.45 requests per second
    SPOP: 71225.07 requests per second
    LPUSH (needed to benchmark LRANGE): 71377.59 requests per second
    LRANGE_100 (first 100 elements): 46125.46 requests per second
    LRANGE_300 (first 300 elements): 23353.57 requests per second
    LRANGE_500 (first 450 elements): 17755.68 requests per second
    LRANGE_600 (first 600 elements): 13846.58 requests per second

### redis-benchmark against nutcracker proxing redis-server

    $ redis-benchmark -h <machine-A> -q -t set,get,incr,lpush,lpop,sadd,spop,lpush,lrange -r 16384 -c 100 -p 22121
    SET: 66622.25 requests per second
    GET: 62814.07 requests per second
    INCR: 64267.35 requests per second
    LPUSH: 70224.72 requests per second
    LPOP: 71428.57 requests per second
    SADD: 69832.40 requests per second
    SPOP: 72411.30 requests per second
    LPUSH (needed to benchmark LRANGE): 69589.42 requests per second
    LRANGE_100 (first 100 elements): 45167.12 requests per second
    LRANGE_300 (first 300 elements): 21556.37 requests per second
    LRANGE_500 (first 450 elements): 15057.97 requests per second
    LRANGE_600 (first 600 elements): 11990.41 requests per second

### redis-benchmark against rcproxy proxing redis-server

    $ redis-benchmark -h <machine-A> -q -t set,get,incr,lpush,lpop,sadd,spop,lpush,lrange -r 16384 -c 100 -p 9736
    SET: 63897.76 requests per second
    GET: 67294.75 requests per second
    INCR: 69204.15 requests per second
    LPUSH: 70771.41 requests per second
    LPOP: 67796.61 requests per second
    SADD: 70621.47 requests per second
    SPOP: 70126.23 requests per second
    LPUSH (needed to benchmark LRANGE): 70921.98 requests per second
    LRANGE_100 (first 100 elements): 39001.56 requests per second
    LRANGE_300 (first 300 elements): 24956.33 requests per second
    LRANGE_500 (first 450 elements): 19022.26 requests per second
    LRANGE_600 (first 600 elements): 15030.81 requests per second
