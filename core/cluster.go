// Copyright (c) 2022 The rcproxy Authors
// Copyright (c) 2011 Twitter, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License"): you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package core

import (
	"bytes"
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"

	"github.com/cornelk/hashmap"
	"github.com/pkg/errors"

	"rcproxy/core/pkg/logging"
	"rcproxy/core/pkg/redis"
)

const (
	LinkStatusConnected    = "connected"
	LinkStatusDisconnected = "disconnected"
)

type Role uint8

const (
	Master Role = iota
	Slave
)

type RedisWrapper interface {
	Dial(address, passwd string, options ...redis.DialOption) (redis.Conn, error)
}

type redisWrapper struct{}

func (s *redisWrapper) Dial(address, passwd string, options ...redis.DialOption) (redis.Conn, error) {
	return redis.Dial(address, passwd, options...)
}

type ClusterNodes struct {
	ServerMap   hashmap.HashMap
	Replicasets []*replicaset

	redisWrapper    RedisWrapper
	redisAddrs      string
	passwd          string
	lastServerNames string
	serverChanged   bool
}

type ClusterNode struct {
	// Name hex string, sha1-size
	Name string
	// Addr ip:Port
	Addr string
	// Ip latest known Ip address of this node
	Ip string
	// Port latest known Port of this node
	Port int
	// CPort latest known Cluster Port of this node
	CPort int
	// Role master or slave
	Role Role
	// MasterId master which slave of
	MasterId string
	// Flags like myself、master、slave、fail?、fail、handshake、noaddr
	Flags string
	// PingSent unix time we sent latest ping
	PingSent int64
	// Unix time we received the pong
	PongReceived int64
	// ConfigEpoch last epoch observed for this node
	ConfigEpoch uint64
	// Connected TCP/IP link with this node, true connected、false disconnected
	Connected bool
	// Version redis node version
	Version string
	// Slots handled by this node
	Slots []Slots
}

type replicaset struct {
	Master *ClusterNode
	Slaves []*ClusterNode
}

type Slots struct {
	Start int32
	End   int32
}

func (c *ClusterNodes) loopClusterNodes() {
	for {
		select {
		case msg := <-EngineGlobal.clusterChan:
			if len(msg) < 3 {
				return
			}
			if msg[0] == '+' && msg[1] == 'O' && msg[2] == 'K' {
				return
			}
			if msg[0] == '$' && msg[1] == '-' && msg[2] == '1' {
				return
			}

			length, err := parseLen(msg[1 : bytes.IndexByte(msg, '\n')-1])
			if err != nil {
				logging.Errorf("[cluster loop] update cluster nodes: nodes info invalid: %s", err)
				return
			}
			if length > 163840 {
				logging.Errorf("[cluster loop] update cluster nodes: nodes info too large > 163840")
				return
			}

			if err := c.updateClusterNodes(string(msg[bytes.IndexByte(msg, '\n')+1 : len(msg)-3])); err != nil {
				logging.Errorf("[cluster loop] update cluster nodes err: %s", err)
			}
		}
	}
}

func (c *ClusterNodes) updateClusterNodes(msg string) error {
	allNodes, err := c.parse(msg)
	if err != nil {
		logging.Errorf("[cluster loop] conn.CLusterNodes error: %s", err)
		return errors.Wrapf(err, "redis do cluster nodes error")
	}

	if c.isChanged(allNodes) {
		c.setServer(allNodes)
		c.setReplicaset(allNodes)
		c.serverChanged = true
	}

	return nil
}

func (c *ClusterNodes) isChanged(allNodes []*ClusterNode) (changed bool) {
	changed = false
	if len(allNodes) != c.ServerMap.Len() {
		changed = true
	}

	var serverNames []string
	for _, n := range allNodes {
		if n.Role == Master {
			serverNames = append(serverNames, fmt.Sprintf("%s#%d#%v", n.Addr, n.Role, n.Slots))
		} else {
			serverNames = append(serverNames, fmt.Sprintf("%s#%d", n.Addr, n.Role))
		}
	}
	sort.Strings(serverNames)
	tmpServerNames := strings.Join(serverNames, ",")
	if tmpServerNames != c.lastServerNames {
		changed = true
		logging.Infof("[cluster loop] servers changes detected")
		logging.Infof("[cluster loop] lastServers: len=%d %s", c.ServerMap.Len(), c.lastServerNames)
		logging.Infof("[cluster loop] newServers: len=%d %s", len(allNodes), tmpServerNames)
	}
	c.lastServerNames = tmpServerNames
	return changed
}

func (c *ClusterNodes) setServer(allNodes []*ClusterNode) {
	for kv := range c.ServerMap.Iter() {
		c.ServerMap.Del(kv.Key)
	}
	for _, m := range allNodes {
		c.ServerMap.Insert(m.Addr, m)
	}
	logging.Infof("[cluster loop] set server done")
	return
}

func (c *ClusterNodes) setReplicaset(allNodes []*ClusterNode) {
	if c.Replicasets == nil {
		c.Replicasets = make([]*replicaset, 0)
	}
	c.Replicasets = c.Replicasets[:0]

	for _, n := range allNodes {
		if n.Role == Master {
			r := new(replicaset)
			r.Master = n
			c.Replicasets = append(c.Replicasets, r)
		}
	}
	for _, n := range allNodes {
		if n.Role == Slave {
			for _, rs := range c.Replicasets {
				if rs.Master.Name == n.MasterId {
					rs.Slaves = append(rs.Slaves, n)
					break
				}
			}
		}
	}
	logging.Infof("[cluster loop] set replicaset done")
	return
}

func (c *ClusterNodes) parse(msgs string) (allNodes []*ClusterNode, err error) {
	lines := strings.Split(msgs, string('\n'))
	for _, line := range lines {
		xs := strings.Split(line, " ")
		if len(xs) < 8 {
			logging.Warnf("[cluster loop] skip redis node because lack of column, line: %+v", xs)
			continue
		}
		if strings.Contains(xs[2], "noaddr") || strings.Contains(xs[2], "handshake") {
			logging.Warnf("[cluster loop] skip redis node because the flag marked as noaddr or handshake, line: %+v", xs)
			continue
		}
		if strings.Contains(xs[2], "fail") {
			logging.Warnf("[cluster loop] skip redis node because the flag marked as fail, line: %+v", xs)
			continue
		}
		if !strings.Contains(xs[2], "master") && !strings.Contains(xs[2], "slave") {
			logging.Warnf("[cluster loop] skip redis node because the flag is neither master or slave, line: %+v", xs)
			continue
		}
		if strings.Contains(xs[7], "disconnected") {
			logging.Warnf("[cluster loop] skip redis node because of disconnected, line: %+v", xs)
			continue
		}

		node, err := c.newClusterNode(xs)
		if err != nil {
			logging.Warnf("[cluster loop] skip redis node because of error occurred, err: %s, line: %+v", err, xs)
			continue
		}

		_, ok := c.ServerMap.Get(node.Addr)
		if !ok {
			info, err := c.redisInfo(node.Addr)
			if err != nil {
				logging.Warnf("[cluster loop] skip redis node because of info command error occurred, err: %s, line: %+v", err, xs)
				continue
			}
			if node.Role == Slave && info.Loading {
				logging.Warnf("[cluster loop] skip redis node because of slave loading, node: %+v, info: %+v", node, info)
				continue
			}
			if node.Role == Slave && info.MasterLinkStatus != "up" {
				logging.Warnf("[cluster loop] skip redis node because of slave master_link_status down, node: %+v, info: %+v", node, info)
				continue
			}
			node.Version = info.Version
		}

		allNodes = append(allNodes, node)
	}

	if len(allNodes) < 3 {
		return nil, errors.New("not enough nodes")
	}

	return allNodes, nil
}

func (c *ClusterNodes) newClusterNode(line []string) (*ClusterNode, error) {
	node := new(ClusterNode)
	node.Name = line[0]
	node.Addr, node.Ip, node.Port, node.CPort = node.parseAddr(line[1])
	if len(node.Addr) < 1 {
		return nil, errors.Errorf("node %s addr invalid", node.Addr)
	}
	if strings.Contains(line[2], "master") {
		node.Role = Master
	} else {
		node.Role = Slave
	}
	node.Flags = line[2]
	node.MasterId = line[3]
	node.PingSent, _ = strconv.ParseInt(line[4], 10, 64)
	node.PongReceived, _ = strconv.ParseInt(line[5], 10, 64)
	node.ConfigEpoch, _ = strconv.ParseUint(line[6], 10, 64)
	node.Connected = line[7] == LinkStatusConnected

	if node.Role == Slave {
		return node, nil
	}
	if len(line) < 9 {
		return nil, errors.New("slot not found")
	}

	for i := 8; i < len(line); i++ {
		if strings.HasPrefix(line[i], "[") {
			continue
		}
		start, end, err := node.parseSlot(line[i])
		if err != nil {
			return nil, err
		}
		node.Slots = append(node.Slots, Slots{start, end})
	}
	return node, nil
}

// @input 127.0.0.1:6379@16379
// @output ipAndPort 127.0.0.1:6379
// @output ip 127.0.0.1
// @output port 6379
// @output cport 16379
func (c *ClusterNode) parseAddr(addrStr string) (string, string, int, int) {
	var parseIPAndPorts = func(str string) (string, string) {
		addr := strings.Split(str, ":")
		if len(addr) <= 1 {
			return "", ""
		}
		return addr[0], addr[1]
	}

	var parsePortAndCPort = func(str string) (string, string) {
		ports := strings.Split(str, "@")
		if len(ports) <= 1 {
			return ports[0], ""
		}
		return ports[0], ports[1]
	}

	ip, portAndCPort := parseIPAndPorts(addrStr)
	if len(ip) < 1 {
		logging.Errorf("[cluster loop] invalid %s redis address from command `cluster nodes`", addrStr)
		return "", "", 0, 0
	}

	portStr, cPortStr := parsePortAndCPort(portAndCPort)
	if len(portStr) < 1 {
		logging.Errorf("[cluster loop] invalid %s redis address from command `cluster nodes`", addrStr)
		return "", "", 0, 0
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		logging.Warnf("[cluster loop] invalid %s redis port from command `cluster nodes`", addrStr)
		return "", "", 0, 0
	}

	cPort := 0
	if len(cPortStr) > 0 {
		cPort, _ = strconv.Atoi(cPortStr)
	}
	return net.JoinHostPort(ip, portStr), ip, port, cPort
}

func (c *ClusterNode) parseSlot(slotsStr string) (int32, int32, error) {

	var err error
	var start, end int64
	slots := strings.Split(slotsStr, "-")
	start, err = strconv.ParseInt(slots[0], 10, 32)
	if err != nil {
		return -1, -1, errors.New("slot parse failed")
	}
	if len(slots) <= 1 {
		return int32(start), int32(start), nil
	}
	end, err = strconv.ParseInt(slots[1], 10, 32)
	if err != nil {
		return -1, -1, errors.New("slot parse failed")
	}
	return int32(start), int32(end), nil
}

func (c *ClusterNodes) redisInfo(addr string) (*redis.Info, error) {
	conn, err := c.redisWrapper.Dial(addr, c.passwd)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	return conn.Info()
}

func GetClusterNodes() []*ClusterNode {
	var nodes []*ClusterNode
	for kv := range EngineGlobal.ClusterNodes.ServerMap.Iter() {
		nodes = append(nodes, kv.Value.(*ClusterNode))
	}
	return nodes
}
