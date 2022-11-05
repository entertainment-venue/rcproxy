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
	"net/http"

	"github.com/gin-gonic/gin"

	"rcproxy/core"
)

type ClusterNodeRes struct {
	*core.ClusterNode
	Slavers []*core.ClusterNode
}

func HandleClusters(c *gin.Context) {
	var res []*ClusterNodeRes
	clusterNodes := core.GetClusterNodes()
	for _, node := range clusterNodes {
		if node.Role == core.Master {
			res = append(res, &ClusterNodeRes{ClusterNode: node})
		}
	}
	for _, node := range clusterNodes {
		if node.Role == core.Slave {
			for _, mNode := range res {
				if node.MasterId == mNode.Name {
					mNode.Slavers = append(mNode.Slavers, node)
				}
			}
		}
	}

	c.JSON(http.StatusOK, res)
}
