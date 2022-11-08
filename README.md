English | [中文](README_ZH.md)

# rcproxy (Redis Cluster Proxy)

## Introduction

`rcproxy` as the name implies, provides a proxy for `Redis Cluster`.

`rcproxy` is based on `gnet` network framework, and uses a single thread/goroutine network model with a high-performance event loop. Test performance is comparable to `twemproxy`.

`rcproxy` inspired by `twemproxy`.

## Features

- [x] High-performance event-loop under networking model of single thread/goroutine.
- [x] Automatically shards to multiple redis nodes.
- [x] Keeps a small number of connections to redis.
- [x] Detects health status of `Redis cluster` every second.
- [x] Implements the complete redis protocol.
- [x] Read/Write Splitting, and load balancing between slaves.
- [x] Supports IP whitelist dynamic loading.
- [x] Supports `Prometheus Metrics` endpoint, exposure observation metrics.
- [x] Verified in Redis 3.0.3/6.2.6
- [x] Works with Linux, OS X.

## Getting Started

* go >= 1.17

```bash
git clone git@github.com:entertainment-venue/rcproxy.git
cd rcproxy && make
./bin/rcproxy -p conf -c yc.yaml -a authip.yaml
```

## Details of rcproxy

* [Redis Command Support](./docs/command.md)
* [Performance](./docs/performance.md)
* [Rcproxy Endpoints](./docs/endpoints.md)

## License

Source code of `rcproxy` should be distributed under the Apache-2.0 license.

## Acknowledgments

* [gnet](https://github.com/panjf2000/gnet)
* [twemproxy](https://github.com/twitter/twemproxy)