// Copyright (c) 2022 The rcproxy Authors
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

package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"path"
	"syscall"

	"github.com/gin-gonic/gin"

	"rcproxy/config"
	"rcproxy/core"
	"rcproxy/core/authip"
	"rcproxy/core/pkg/logging"
	"rcproxy/core/server"
	"rcproxy/web"
)

var (
	configPath       = flag.String("p", "conf", "Config file path")
	basicConfigFile  = flag.String("c", "rc.yaml", "Basic config filename")
	authIpConfigFile = flag.String("a", "authip.yaml", "Authip config filename")
	version          = flag.Bool("v", false, "Show version")
	help             = flag.Bool("h", false, "Show usage info")
)

var (
	CommitSHA string
	Tag       string
	BuildTime string
)

func init() {
	if len(Tag) < 1 {
		Tag = "unknown"
	}
	if len(CommitSHA) < 1 {
		CommitSHA = "unknown"
	}
	if len(BuildTime) < 1 {
		BuildTime = "unknown"
	}
}

const banner string = `
___________________________________________  ___  __
___  __ \_  ____/__  __ \__  __ \_  __ \_  |/ / \/ /
__  /_/ /  /    __  /_/ /_  /_/ /  / / /_    /__  / 
_  _, _// /___  _  ____/_  _, _// /_/ /_    | _  /  
/_/ |_| \____/  /_/     /_/ |_| \____/ /_/|_| /_/   
                                                    
`

func parseCli() {
	flag.Parse()
	if *version {
		fmt.Printf("version: %s\ncommit: %s\ntime: %s\n", Tag, CommitSHA, BuildTime)
		os.Exit(0)
	}
	if *help {
		flag.Usage()
		os.Exit(0)
	}
}

func main() {
	parseCli()

	cfg, err := config.LoadConfig(path.Join(*configPath, *basicConfigFile))
	if err != nil {
		logging.Errorf("parse config file err:%v", err)
		return
	}

	// Initialization Logger
	if err = logging.InitializeLogger(
		logging.WithPath(cfg.LogPath),
		logging.WithExpireDay(cfg.LogExpireDay),
		logging.WithLogLevel(cfg.LogLevel),
	); err != nil {
		logging.Errorf("failed to initialize logger, err: %s", err)
		return
	}

	fmt.Print(banner)
	fmt.Printf("rcproxy version: %s\n", Tag)
	fmt.Printf("rcproxy started with port: %d, pid: %d\n", cfg.Port, syscall.Getpid())
	logging.Infof("rcproxy started with port: %d, pid: %d, rcproxy version: %s", cfg.Port, syscall.Getpid(), Tag)

	// Only whitelisted addresses can access redis
	if err := authip.LoopIPWhiteList(*configPath, *authIpConfigFile); err != nil {
		logging.Errorf("failed to loop IP white list, err: %s", err)
		return
	}

	if cfg.WebPort > 0 {
		// Initialization http server
		addr := fmt.Sprintf(":%d", cfg.WebPort)
		gin.SetMode(gin.ReleaseMode)
		ginSrv := gin.New()
		web.Init(ginSrv)
		httpSrv := &http.Server{Handler: ginSrv, Addr: addr}
		go func() {
			if err = httpSrv.ListenAndServe(); err != nil {
				logging.Errorf("failed to start http server, err: %s", err)
				return
			}
		}()
	}

	tcpServer := server.NewListenServer(
		server.WithRedisPassword(cfg.Redis.Password),
		server.WithServerRetryTimeout(cfg.Redis.ServerRetryTimeout),
		server.WithDisableRedisSlave(cfg.Redis.DisableSlave),
	)
	if err = core.Run(
		tcpServer,
		fmt.Sprintf("tcp://:%d", cfg.Port),
		core.WithRedisPasswd(cfg.Redis.Password),
		core.WithRedisServers(cfg.Redis.Servers),
		core.WithRedisPreconnect(cfg.Redis.Preconnect),
		core.WithRedisConnectTimeout(cfg.Redis.ConnTimeout),
		core.WithRedisRequestTimeout(cfg.Redis.Timeout),
		core.WithRedisServerConnections(cfg.Redis.ServerConnections),
		core.WithRedisMsgMaxLength(cfg.Redis.MsgMaxLengthLimit),
		core.WithSlowlogSlowerThan(cfg.Redis.SlowlogSlowerThan),
	); err != nil {
		logging.Errorf("rcproxy run failed: %s", err)
	}

	logging.Infof("rcproxy shutdown, pid: %d, listen: %d", syscall.Getpid(), cfg.Port)
}
