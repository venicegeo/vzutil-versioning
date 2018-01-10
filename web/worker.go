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
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"log"
	"os/exec"
	"regexp"
	"strings"

	"github.com/venicegeo/pz-gocommon/elasticsearch"
)

type worker struct {
	index *elasticsearch.Index
	queue chan GitWebhook
}

func NewWorker(i *elasticsearch.Index) *worker {
	wrkr := worker{i, make(chan GitWebhook)}
	return &wrkr
}

var depRe = regexp.MustCompile(`###   (.+):(.+):(.+):(.+)`)

func (w *worker) Start() {
	go func() {
		for {
			git := <-w.queue
			dat, err := exec.Command("./single", git.Repository.FullName, git.AfterSha).Output()
			if err != nil {
				log.Println("Unable to run against", git.Repository.FullName, git.AfterSha, ":", err.Error())
				continue
			}
			parts := strings.Split(string(dat), "\n")[2:]
			deps := []Dependency{}
			for _, p := range parts {
				if p == "" {
					continue
				}
				matches := depRe.FindStringSubmatch(p)
				deps = append(deps, Dependency{matches[1], matches[2], matches[4]})
			}
			for _, d := range deps {
				fmt.Println(d.GetHashSum())
			}
		}
	}()
}

func (w *worker) AddTask(git GitWebhook) {
	w.queue <- git
}

type Dependency struct {
	Name     string `json:"name"`
	Version  string `json:"version"`
	Language string `json:"language"`
}

func (d *Dependency) GetHashSum() string {
	tmp := md5.Sum([]byte(fmt.Sprintf("%s:%s:%s", d.Name, d.Version, d.Language)))
	return hex.EncodeToString(tmp[:])
}
