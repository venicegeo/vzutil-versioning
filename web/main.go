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
	fmt.Println("Starting up...")

	url, user, pass, err := getVcapES()
	fmt.Printf("The elasticsearch url has been found to be [%s]\n", url)
	if user != "" {
		fmt.Println("There is a username")
	}
	if pass != "" {
		fmt.Println("There is a password")
	}
	if err != nil {
		log.Fatal(err)
	}
	i, err := elasticsearch.NewIndex2(url, user, pass, "test", `
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
		//	log.Fatal(err.Error())
	} else {
		fmt.Println(i.GetVersion())
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "20012"
	}

	fmt.Println("Starting on port", port)
	server := Server{}
	server.Configure([]RouteData{RouteData{"GET", "/", defaultPath},
		RouteData{"POST", "/webhook", webhookPath}})
	err = <-server.Start(":" + port)
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
	err := c.BindJSON(&obj)
	if err != nil {
		log.Println("Unable to bind")
		c.Status(400)
		return
	}
	git, ok := obj.(GitWebhook)
	if !ok {
		log.Println("Unable to convert to webhook struct")
		c.Status(400)
		return
	}
	fmt.Println(git.Repository.FullName, git.AfterSha)
	c.Status(200)
}
