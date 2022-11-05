[英文](README.md) | 中文

# rcproxy (Redis Cluster Proxy)

## 简介

`rcproxy`，顾名思义，为`Redis Cluster`模式提供代理。

`rcproxy`基于`gnet`网络框架开发，使用了单线/协程的事件轮询网络模型，极致的压榨CPU单核性能，实测性能与`twemproxy`相媲美。

## 功能

- [x] 高性能，基于单线程（或单协程）网络模型开发
- [x] 自动分片至多个后端Redis节点
- [x] 与后端`Redis`节点保持极少的连接数
- [x] 每秒探测`Redis`集群健康状态（基于`cluster nodes`命令），及时屏蔽不健康节点
- [x] 支持完整的`Redis RESP`协议，支持大部分`Redis`命令
- [x] 读写分离，从库负载均衡
- [x] IP白名单，支持热加载
- [x] 支持`Prometheus Metrics`接口，暴露观察指标
- [x] 支持**Linux/OS X**多种平台

## 开始

* go >= 1.17

```bash
git clone git@github.com:entertainment-venue/rcproxy.git
cd rcproxy && make
./bin/rcproxy -p conf -c yc.yaml -a authip.yaml
```

## 证书

`rcproxy` 的源码需在遵循 Apache-2.0 开源证书的前提下使用。

## 鸣谢

- 感谢[gnet](https://github.com/panjf2000/gnet)，rcproxy基于gnet网络框架二次开发。
- 感谢[twemproxy](https://github.com/twitter/twemproxy)，rcproxy大量借鉴了twemproxy设计。