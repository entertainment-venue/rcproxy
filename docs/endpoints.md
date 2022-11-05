# Rcproxy Endpoints

## Notice
- Must specify `web_port` configuration

## Catalog
- [View rcproxy version](#version)
- [View ip whitelist](#authip)
- [View healthy cluster nodes](#health_nodes)
- [View metrics](#metrics)

<h3 id="version">View rcproxy version</h3>

```
Action: GET
URL: http://127.0.0.1:9797/version
```
#### Example
```
curl -X GET http://127.0.0.1:9737/version

{
    "CommitSHA":"",
    "Tag":"1.0.0",
    "BuildTime":"2022-11-05 11:20:56"
}
```

<h3 id="authip">View ip whitelist</h3>

```
Action: GET
URL: http://127.0.0.1:9797/authip
```
#### Example
```
curl -X GET http://127.0.0.1:9737/authip

[
    "127.0.0.1",
    "127.0.0.2",
    "127.0.0.3"
]
```

<h3 id="health_nodes">View healthy cluster nodes</h3>

```
Action: GET
URL: http://127.0.0.1:9797/cluster/nodes
```
#### Example
```
curl -X GET http://127.0.0.1:9737/cluster/nodes

[
    {
        "Name":"4a28a6d3fa678731c3241db99056063248121230",
        "Addr":"127.0.0.1:8330",
        "Ip":"127.0.0.1",
        "Port":8330,
        "CPort":18330,
        "Role":0,
        "MasterId":"-",
        "Flags":"master",
        "PingSent":0,
        "PongReceived":1667409164461,
        "ConfigEpoch":10,
        "Connected":true,
        "Version":"6.2.6",
        "Slots":[
            {
                "Start":0,
                "End":5460
            }
        ],
        "Slavers":[
            {
                "Name":"5a2d74cc20d2c38192f80f55cc16bb5651296834",
                "Addr":"127.0.0.2:8330",
                "Ip":"127.0.0.2",
                "Port":8330,
                "CPort":18330,
                "Role":1,
                "MasterId":"4a28a6d3fa678731c3241db99056063248121230",
                "Flags":"slave",
                "PingSent":0,
                "PongReceived":1667409162000,
                "ConfigEpoch":10,
                "Connected":true,
                "Version":"6.2.6",
                "Slots":null
            },
            {
                "Name":"885e47d4d54bb403ed544f16972e531dfcefaad6",
                "Addr":"127.0.0.3:8330",
                "Ip":"127.0.0.3",
                "Port":8330,
                "CPort":18330,
                "Role":1,
                "MasterId":"4a28a6d3fa678731c3241db99056063248121230",
                "Flags":"slave",
                "PingSent":0,
                "PongReceived":1667409162454,
                "ConfigEpoch":10,
                "Connected":true,
                "Version":"6.2.6",
                "Slots":null
            }
        ]
    },
    {
        "Name":"56bf8d2aa06757b2c7ff025d1ef084012a32fe33",
        "Addr":"127.0.0.4:8330",
        "Ip":"127.0.0.4",
        "Port":8330,
        "CPort":18330,
        "Role":0,
        "MasterId":"-",
        "Flags":"master",
        "PingSent":0,
        "PongReceived":1667409158000,
        "ConfigEpoch":2,
        "Connected":true,
        "Version":"6.2.6",
        "Slots":[
            {
                "Start":5461,
                "End":5641
            },
            {
                "Start":5643,
                "End":10922
            }
        ],
        "Slavers":[
            {
                "Name":"ac3a6a161a49e3e216ea74294b85da18d88363b8",
                "Addr":"127.0.0.5:8330",
                "Ip":"127.0.0.5",
                "Port":8330,
                "CPort":18330,
                "Role":1,
                "MasterId":"56bf8d2aa06757b2c7ff025d1ef084012a32fe33",
                "Flags":"slave",
                "PingSent":0,
                "PongReceived":1667409162000,
                "ConfigEpoch":2,
                "Connected":true,
                "Version":"6.2.6",
                "Slots":null
            },
            {
                "Name":"2b741eb189f1c675d766ea147487f65f7cc4e922",
                "Addr":"127.0.0.6:8330",
                "Ip":"127.0.0.6",
                "Port":8330,
                "CPort":18330,
                "Role":1,
                "MasterId":"56bf8d2aa06757b2c7ff025d1ef084012a32fe33",
                "Flags":"slave",
                "PingSent":0,
                "PongReceived":1667409160447,
                "ConfigEpoch":2,
                "Connected":true,
                "Version":"6.2.6",
                "Slots":null
            }
        ]
    },
    {
        "Name":"4e2a84dc28a5aff785699192515d3795e10cd27d",
        "Addr":"127.0.0.7:8330",
        "Ip":"127.0.0.7",
        "Port":8330,
        "CPort":18330,
        "Role":0,
        "MasterId":"-",
        "Flags":"myself,master",
        "PingSent":0,
        "PongReceived":1667409159000,
        "ConfigEpoch":12,
        "Connected":true,
        "Version":"6.2.6",
        "Slots":[
            {
                "Start":5642,
                "End":5642
            },
            {
                "Start":10923,
                "End":16383
            }
        ],
        "Slavers":[
            {
                "Name":"aa5260715cd749f1368c7a06747730daa848fe20",
                "Addr":"127.0.0.8:8330",
                "Ip":"127.0.0.8",
                "Port":8330,
                "CPort":18330,
                "Role":1,
                "MasterId":"4e2a84dc28a5aff785699192515d3795e10cd27d",
                "Flags":"slave",
                "PingSent":0,
                "PongReceived":1667409162000,
                "ConfigEpoch":12,
                "Connected":true,
                "Version":"6.2.6",
                "Slots":null
            },
            {
                "Name":"bdf2c6e052d6f73d25b3f12c0e2b4f27bfc8526f",
                "Addr":"127.0.0.9:8330",
                "Ip":"127.0.0.9",
                "Port":8330,
                "CPort":18330,
                "Role":1,
                "MasterId":"4e2a84dc28a5aff785699192515d3795e10cd27d",
                "Flags":"slave",
                "PingSent":0,
                "PongReceived":1667409163457,
                "ConfigEpoch":12,
                "Connected":true,
                "Version":"6.2.6",
                "Slots":null
            }
        ]
    }
]
```

<h3 id="metrics">View metrics</h3>

```
Action: GET
URL: http://127.0.0.1:9797/metrics
```
#### Example
```
curl -X GET http://127.0.0.1:9737/metrics

# HELP rcproxy_cmd number of redis command requests
# TYPE rcproxy_cmd counter
rcproxy_cmd{cmd="get"} 4
rcproxy_cmd{cmd="hashs"} 5
rcproxy_cmd{cmd="incr_decr"} 2
rcproxy_cmd{cmd="set"} 1
rcproxy_cmd{cmd="string"} 5
# HELP rcproxy_curr_connections current connections
# TYPE rcproxy_curr_connections gauge
rcproxy_curr_connections{type="client"} 0
rcproxy_curr_connections{type="server"} 9
rcproxy_curr_connections{type="total"} 9
# HELP rcproxy_redis_connections_active number of active connections between proxy and redis
# TYPE rcproxy_redis_connections_active gauge
rcproxy_redis_connections_active{addr="127.0.0.1:8300"} 1
rcproxy_redis_connections_active{addr="127.0.0.1:8302"} 1
rcproxy_redis_connections_active{addr="127.0.0.1:8304"} 1
rcproxy_redis_connections_active{addr="127.0.0.1:8306"} 1
rcproxy_redis_connections_active{addr="127.0.0.1:8308"} 1
rcproxy_redis_connections_active{addr="127.0.0.1:8310"} 1
rcproxy_redis_connections_active{addr="127.0.0.1:8312"} 1
rcproxy_redis_connections_active{addr="127.0.0.1:8314"} 1
rcproxy_redis_connections_active{addr="127.0.0.1:8316"} 1
# HELP rcproxy_request_latency request latency
# TYPE rcproxy_request_latency histogram
rcproxy_request_latency_bucket{le="10"} 0
rcproxy_request_latency_bucket{le="20"} 0
rcproxy_request_latency_bucket{le="50"} 11
rcproxy_request_latency_bucket{le="100"} 11
rcproxy_request_latency_bucket{le="200"} 11
rcproxy_request_latency_bucket{le="500"} 12
rcproxy_request_latency_bucket{le="+Inf"} 12
rcproxy_request_latency_sum 768
rcproxy_request_latency_count 12
# HELP rcproxy_total_connections total connections
# TYPE rcproxy_total_connections counter
rcproxy_total_connections 11
# HELP rcproxy_total_requests total requests
# TYPE rcproxy_total_requests counter
rcproxy_total_requests 29
```

