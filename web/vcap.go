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
	"encoding/json"
	"os"
)

type Vcap struct {
	UserProvided VcapServices `json:"user-provided"`
	EceProvided  VcapServices `json:"ece"`
}

type VcapServices []VcapService

type VcapService struct {
	Credentials VcapCredentials `json:"credentials"`
	Name        string          `json:"name"`
}

type VcapCredentials struct {
	Password  string `json:"password"`
	Port      string `json:"port"`
	Host      string `json:"host"`
	ClusterId string `json:"clusterId"`
	Uri       string `json:"uri"`
	Username  string `json:"elastic"`
}

func getVcapES() (string, string, string, error) {
	services := os.Getenv("VCAP_SERVICES")
	if services == "" {
		return "http://127.0.0.1:9200", "", "", nil
	}
	var vcap Vcap
	if err := json.Unmarshal([]byte(services), &vcap); err != nil {
		return "", "", "", err
	}
	searchArr := func(services VcapServices) (string, string, string, bool) {
		for _, e := range services {
			if e.Name == "pz-elasticsearch" {
				url := e.Credentials.Uri
				if url == "" {
					url = "http://" + e.Credentials.Host
					if e.Credentials.Port != "" {
						url += ":" + e.Credentials.Port
					}
				}
				return url, e.Credentials.Username, e.Credentials.Password, true
			}
		}
		return "", "", "", false
	}
	url, user, pass, ok := searchArr(vcap.UserProvided)
	if !ok {
		url, user, pass, ok = searchArr(vcap.EceProvided)
	}
	if !ok {
		url = "localhost:9200"
	}
	return url, user, pass, nil
}
