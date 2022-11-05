package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"rcproxy/core/pkg/redis"
)

type mockedRedisWrapper struct {
	mock.Mock
}

type mockedRedis struct {
	mock.Mock
}

func (m *mockedRedis) Do(cmd string, i ...interface{}) (interface{}, error) {
	args := m.Called(cmd, i)
	return args.Get(0), args.Error(1)
}

func (m *mockedRedis) Info() (*redis.Info, error) {
	args := m.Called()
	return args.Get(0).(*redis.Info), args.Error(1)
}

func (m *mockedRedis) Send(string, ...interface{}) error { return nil }
func (m *mockedRedis) Flush() error                      { return nil }
func (m *mockedRedis) Receive() (interface{}, error)     { return nil, nil }
func (m *mockedRedis) Close() error                      { return nil }

func (m *mockedRedisWrapper) Dial(addr, passwd string, _ ...redis.DialOption) (redis.Conn, error) {
	args := m.Called(addr, passwd)
	return args.Get(0).(redis.Conn), args.Error(1)
}

func TestClusterNodes(t *testing.T) {
	mRedis := new(mockedRedis)
	mRedis.On("Info").Return(&redis.Info{Loading: false, MasterLinkStatus: "up", Version: "3.0.2"}, nil)

	wrapper := new(mockedRedisWrapper)
	wrapper.On("Dial", mock.Anything, mock.Anything).Return(mRedis, nil)

	c := ClusterNodes{
		redisAddrs:   "127.0.0.1:6379",
		passwd:       "",
		redisWrapper: wrapper,
	}
	var msg = "00024e4759fc874a55362b9fe7472859cc4235c0 127.0.0.1:8300 myself,master - 0 0 1 connected 0-5460 5643-10922 [5642->-aa5260715cd749f1368c7a06747730daa848fe20]\n01ae6b52c5bcee240275d7b96ee0c33cb4615f01 127.0.0.1:8308 slave 00024e4759fc874a55362b9fe7472859cc4235c0 0 1646637827924 5 connected\nd5c94de92eff84aeab97eaf66079869b0e130f1e 127.0.0.1:8304 master - 0 1646637824420 3 connected 10923-16383\n731aaa0d9dae20695fe7e7702f14d5ad0e10219a 127.0.0.1:8302 master - 0 1646637824921 2 connected 5461-10922\n80651576a8fe05d3eca0678d3b39dc2b0a5315a0 127.0.0.1:8314 slave d5c94de92eff84aeab97eaf66079869b0e130f1e 0 1646637829927 8 connected\n8cb42cde94bf5c906d1696d337b04bb9da3cb205 127.0.0.1:8310 slave 731aaa0d9dae20695fe7e7702f14d5ad0e10219a 0 1646637828926 6 connected\n4008272410c2b3257f9a50a3aaca275220fec990 127.0.0.1:8306 slave 00024e4759fc874a55362b9fe7472859cc4235c0 0 1646637823418 4 connected\n85f4ad4e797ef653cadb943aaab804a7b986ac39 127.0.0.1:8312 slave 731aaa0d9dae20695fe7e7702f14d5ad0e10219a 0 1646637823919 7 connected\nfa565f0baa0e304c945e37242dac8ce46a6a95be 127.0.0.1:8316 slave d5c94de92eff84aeab97eaf66079869b0e130f1e 0 1646637826923 9 connected"
	allNodes, err := c.parse(msg)
	assert.Equal(t, nil, err)
	assert.Equal(t, 9, len(allNodes))
	assert.Equal(t, "00024e4759fc874a55362b9fe7472859cc4235c0", allNodes[0].Name)
	assert.Equal(t, "127.0.0.1:8300", allNodes[0].Addr)
	assert.Equal(t, "127.0.0.1", allNodes[0].Ip)
	assert.Equal(t, 8300, allNodes[0].Port)
	assert.Equal(t, 0, allNodes[0].CPort)
	assert.Equal(t, Master, allNodes[0].Role)
	assert.Equal(t, "-", allNodes[0].MasterId)
	assert.Equal(t, "3.0.2", allNodes[0].Version)
	assert.Equal(t, 2, len(allNodes[0].Slots))
	assert.Equal(t, int32(0), allNodes[0].Slots[0].Start)
	assert.Equal(t, int32(5460), allNodes[0].Slots[0].End)
	assert.Equal(t, int32(5643), allNodes[0].Slots[1].Start)
	assert.Equal(t, int32(10922), allNodes[0].Slots[1].End)
}

func TestClusterNodesDown(t *testing.T) {
	mRedis := new(mockedRedis)
	mRedis.On("Info").Return(&redis.Info{Loading: true, MasterLinkStatus: "up"}, nil)

	wrapper := new(mockedRedisWrapper)
	wrapper.On("Dial", mock.Anything, mock.Anything).Return(mRedis, nil)

	c := ClusterNodes{
		redisAddrs:   "127.0.0.1:6379",
		passwd:       "",
		redisWrapper: wrapper,
	}
	var msg = "00024e4759fc874a55362b9fe7472859cc4235c0 127.0.0.1:8300 myself,master - 0 0 1 connected 0-5460\n01ae6b52c5bcee240275d7b96ee0c33cb4615f01 127.0.0.1:8308 slave,fail 00024e4759fc874a55362b9fe7472859cc4235c0 0 1646637827924 5 connected\nd5c94de92eff84aeab97eaf66079869b0e130f1e 127.0.0.1:8304 master - 0 1646637824420 3 connected 10923-16383\n731aaa0d9dae20695fe7e7702f14d5ad0e10219a 127.0.0.1:8302 master - 0 1646637824921 2 connected 5461-10922\n80651576a8fe05d3eca0678d3b39dc2b0a5315a0 127.0.0.1:8314 slave,?fail d5c94de92eff84aeab97eaf66079869b0e130f1e 0 1646637829927 8 connected\n8cb42cde94bf5c906d1696d337b04bb9da3cb205 127.0.0.1:8310 slave 731aaa0d9dae20695fe7e7702f14d5ad0e10219a 0 1646637828926 6 disconnected\n4008272410c2b3257f9a50a3aaca275220fec990 127.0.0.1:8306 slave 00024e4759fc874a55362b9fe7472859cc4235c0 0 1646637823418 4 connected\n85f4ad4e797ef653cadb943aaab804a7b986ac39 127.0.0.1:8312 slave 731aaa0d9dae20695fe7e7702f14d5ad0e10219a 0 1646637823919 7 connected\nfa565f0baa0e304c945e37242dac8ce46a6a95be 127.0.0.1:8316 slave d5c94de92eff84aeab97eaf66079869b0e130f1e 0 1646637826923 9 connected"

	allNodes, err := c.parse(msg)
	assert.Equal(t, nil, err)
	assert.Equal(t, 3, len(allNodes))
}
