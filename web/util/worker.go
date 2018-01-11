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

package util

import (
	"encoding/json"
	"log"
	"os/exec"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/venicegeo/pz-gocommon/elasticsearch"
	"github.com/venicegeo/vzutil-versioning/web/es"
)

type Worker struct {
	singleLocation string
	index          *elasticsearch.Index
	queue          chan *GitWebhook
	mux            *sync.Mutex
}

func NewWorker(i *elasticsearch.Index, singleLocation string) *Worker {
	wrkr := Worker{singleLocation, i, make(chan *GitWebhook), &sync.Mutex{}}
	return &wrkr
}

var depRe = regexp.MustCompile(`###   (.+):(.+):(.+):(.+)`)

func (w *Worker) Start() {
	go func() {
		for {
			git := <-w.queue
			hashes := []string{}
			var project es.Project
			var projectEntries *es.ProjectEntries
			var exists bool

			dat, err := exec.Command(w.singleLocation, git.Repository.FullName, git.AfterSha).Output()
			if err != nil {
				log.Println("Unable to run against", git.Repository.FullName, git.AfterSha, ":", err.Error())
				continue
			}
			w.mux.Lock()
			{
				deps := []es.Dependency{}
				for _, p := range strings.Split(string(dat), "\n")[2:] {
					if p == "" {
						continue
					}
					matches := depRe.FindStringSubmatch(p)
					deps = append(deps, es.Dependency{matches[1], matches[2], matches[4]})
				}
				for _, d := range deps {
					hash := d.GetHashSum()
					hashes = append(hashes, hash)
					exists, err := w.index.ItemExists("dependency", hash)
					if err != nil || !exists {
						resp, err := w.index.PostData("dependency", hash, d)
						if err != nil {
							log.Println("Unable to create dependency", hash, ":", err.Error())
						} else if !resp.Created {
							log.Println("Unable to create dependency", hash)
						}
					}
				}
			}
			sort.Strings(hashes)
			docName := strings.Replace(git.Repository.FullName, "/", "_", -1)
			if exists, err = w.index.ItemExists("project", docName); err != nil {
				log.Println("Error checking project exists:", err.Error())
				continue
			}
			if exists {
				resp, err := w.index.GetByID("project", docName)
				if err != nil {
					log.Println("Unable to retrieve project:", err.Error())
					continue
				}
				if err = json.Unmarshal([]byte(*resp.Source), &project); err != nil {
					log.Println("Unable to unmarshal source:", err.Error())
					continue
				}
			} else {
				project = *es.NewProject(git.Repository.FullName, git.Repository.Name)
			}
			if projectEntries, err = project.GetEntries(); err != nil {
				log.Println("Unable to get entries:", err.Error())
				continue
			}
			if project.LastSha != "" {
				referenceSha := project.LastSha
				lastEntry := (*projectEntries)[referenceSha]
				if lastEntry.EntryReference != "" {
					referenceSha = lastEntry.EntryReference
					lastEntry = (*projectEntries)[referenceSha]
				}
				sort.Strings(lastEntry.Dependencies)
				if reflect.DeepEqual(hashes, lastEntry.Dependencies) {
					(*projectEntries)[git.AfterSha] = es.ProjectEntry{EntryReference: referenceSha}
				} else {
					(*projectEntries)[git.AfterSha] = es.ProjectEntry{Dependencies: hashes}
				}
			} else {
				(*projectEntries)[git.AfterSha] = es.ProjectEntry{Dependencies: hashes}
			}
			project.SetEntries(projectEntries)
			project.LastSha = git.AfterSha

			if strings.HasPrefix(git.Ref, "refs/tags/") {
				tag := strings.Split(git.Ref, "/")[2]
				mapp, err := project.GetTagShas()
				if err != nil {
					log.Println("Unable to get tag shas:", err.Error())
					continue
				}
				mapp[tag] = git.AfterSha
				if err = project.SetTagShas(mapp); err != nil {
					log.Println("Unable to set tag shas:", err.Error())
					continue
				}
			}

			indexProject := func(data func(string, string, interface{}) (*elasticsearch.IndexResponse, error), method string, checkCreate bool) bool {
				resp, err := data("project", docName, project)
				if err != nil {
					log.Println("unable to", method, "project:", err.Error())
					return true
				} else if !resp.Created && checkCreate {
					log.Println("project was not created")
					return true
				}
				return false
			}
			if !exists { //POST
				if indexProject(w.index.PostData, "post", true) {
					continue
				}
			} else { //PUT
				if indexProject(w.index.PutData, "put", false) {
					continue
				}
			}
			w.mux.Unlock()
		}
	}()
}

func (w *Worker) AddTask(git *GitWebhook) {
	w.queue <- git
}
