// Copyright (c) 2022 The rcproxy Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"io/ioutil"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	"rcproxy/core/pkg/logging"
)

type Config struct {
	Port         int         `yaml:"port"`
	WebPort      int         `yaml:"web_port"`
	LogPath      string      `yaml:"log_path"`
	LogLevel     string      `yaml:"log_level"`
	LogExpireDay int         `yaml:"log_expire_day"`
	Redis        redisConfig `yaml:"redis"`
}

type redisConfig struct {
	Servers            string `yaml:"servers"`
	Password           string `yaml:"password"`
	DisableSlave       bool   `yaml:"disable_slave"`
	Preconnect         bool   `yaml:"preconnect"`
	MsgMaxLengthLimit  int    `yaml:"msg_max_length_limit"`
	ConnTimeout        int    `yaml:"conn_timeout"`
	Timeout            int    `yaml:"timeout"`
	ServerRetryTimeout int    `yaml:"server_retry_timeout"`
	ServerConnections  int    `yaml:"server_connections"`
	SlowlogSlowerThan  int64  `yaml:"slowlog_slower_than"`
}

func LoadConfig(fileName string) (*Config, error) {
	file, err := ioutil.ReadFile(fileName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read file from %s", fileName)
	}
	var cfg Config
	if err = yaml.Unmarshal(file, &cfg); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal config from %s", fileName)
	}
	if err = cfg.validate(); err != nil {
		return nil, errors.Wrapf(err, "config validate failed")
	}
	return &cfg, nil
}

func (c *Config) validate() error {
	if v, ok := logging.LevelMapperRev[c.LogLevel]; !ok {
		return errors.Errorf("unknown log level %s", v)
	}
	if len(c.Redis.Servers) < 1 {
		return errors.Errorf("unknown redis addrs")
	}
	return nil
}
