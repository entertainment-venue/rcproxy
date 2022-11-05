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

package authip

import (
	"io/ioutil"
	"path"

	"github.com/cornelk/hashmap"
	"github.com/fsnotify/fsnotify"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	"rcproxy/core/pkg/logging"
)

type AuthIp struct {
	path string
	name string
}

var IpMap ipMap

type ipMap struct {
	enable bool
	hashmap.HashMap
}

func (i *ipMap) Validate(ip string) bool {
	if i.enable {
		if _, ok := i.Get(ip); !ok {
			return false
		}
	}
	return true
}

func (i *ipMap) Insert(key string, value struct{}) bool {
	_, ok := i.HashMap.GetOrInsert(key, value)
	return ok
}

type authIp struct {
	Enable bool     `yaml:"enable"`
	IpList []string `yaml:"ip_white_list"`
}

func LoopIPWhiteList(confPath, confName string) error {
	a := &AuthIp{
		path: confPath,
		name: path.Join(confPath, confName),
	}
	if err := a.parseAuthIp(); err != nil {
		return err
	}
	return a.watchYml()
}

func (a *AuthIp) watchYml() error {
	watch, err := fsnotify.NewWatcher()
	if err != nil {
		logging.Errorf("err=%s", err)
		return err
	}
	err = watch.Add(a.path)
	if err != nil {
		logging.Errorf("err=%s", err)
		return err
	}
	go func() {
		for {
			select {
			case ev := <-watch.Events:
				if ev.Name == a.name {
					switch {
					case ev.Op&fsnotify.Write == fsnotify.Write:
						fallthrough
					case ev.Op&fsnotify.Rename == fsnotify.Rename:
						if err := a.parseAuthIp(); err != nil {
							logging.Errorf("parser auth ip err: %s", err)
						}
					}
				}
			case err := <-watch.Errors:
				logging.Errorf("err=%s", err)
				return
			}
		}
	}()
	return nil
}

func (a *AuthIp) parseAuthIp() error {
	file, err := ioutil.ReadFile(a.name)
	if err != nil {
		return errors.Wrapf(err, "failed to read file from %s", a.name)
	}
	var auth authIp
	if err := yaml.Unmarshal(file, &auth); err != nil {
		return errors.Wrapf(err, "failed to unmarshal config from %s", a.name)
	}

	IpMap.enable = auth.Enable

	if !IpMap.enable {
		return nil
	}

	for _, ip := range auth.IpList {
		if !IpMap.Insert(ip, struct{}{}) {
			logging.Debugf("set ip %s", ip)
		}
	}
	return nil
}
