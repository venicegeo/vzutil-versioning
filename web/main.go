// Copyright 2018, RadiantBlue Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/venicegeo/pz-gocommon/elasticsearch"
)

func main() {
	fmt.Println("hello")

	i, err := elasticsearch.NewIndex2("http://127.0.0.1:9200", "test", `
{
	"mappings": {
		"project":{
			"dynamic":"strict",
			"properties":{
				"location":{"type":"string"},
				"history": {
					"dynamic":"strict",
					"properties":{
						"sha":{"type":"string"}
					}
				}
			}
		}
	}
}`)
	if err != nil {
		log.Fatal(err.Error())
	}
	fmt.Println(i.GetVersion())

	server := Server{}
	server.Configure([]RouteData{RouteData{"GET", "/", defaultPath},
		RouteData{"POST", "/webhook", webhookPath}})
	err = <-server.Start("127.0.0.1:20009")
	fmt.Println(err)
}

func addFSifMissing(url string) string {
	if !strings.HasSuffix(url, "/") {
		url += "/"
	}
	return url
}

func getRequiredEnv(env string) string {
	temp := os.Getenv(env)
	if temp == "" {
		log.Fatal("Missing env var", env)
	}
	return temp
}

func defaultPath(c *gin.Context) {
	c.String(200, "Welcome to the dependency service!")
}
func webhookPath(c *gin.Context) {
	var obj interface{}
	c.BindJSON(&obj)
	fmt.Println(obj)
	c.Status(200)
}
