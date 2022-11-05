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

package web

import (
	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func Init(ginSrv *gin.Engine) {
	pprof.Register(ginSrv)
	ginSrv.GET("/cluster/nodes", HandleClusters)
	ginSrv.GET("/authip", HandleAuthIp)
	ginSrv.GET("/version", HandleVersion)
	ginSrv.GET("/metrics", gin.WrapH(promhttp.Handler()))
}
