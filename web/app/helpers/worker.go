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

package helpers

import (
	"bufio"
	"log"
	"os"
	"os/exec"
	"reflect"
	"regexp"
	"sort"
	"strings"

	"github.com/venicegeo/pz-gocommon/elasticsearch"
	s "github.com/venicegeo/vzutil-versioning/web/app/structs"
	"github.com/venicegeo/vzutil-versioning/web/es"
	u "github.com/venicegeo/vzutil-versioning/web/util"
)

type Worker struct {
	singleLocation string
	index          *elasticsearch.Index

	numWorkers      int
	checkExistQueue chan *s.GitWebhook
	cloneQueue      chan *s.GitWebhook
	esQueue         chan *work

	diffMan *DifferenceManager

	logFile   *os.File
	logWriter *bufio.Writer
}

type work struct {
	fullName string
	name     string
	sha      string
	ref      string
	hashes   []string
	reall    bool
}

func NewWorker(i *elasticsearch.Index, singleLocation string, numWorkers int, diffMan *DifferenceManager) *Worker {
	wrkr := Worker{singleLocation, i, numWorkers, make(chan *s.GitWebhook, 1000), make(chan *s.GitWebhook, 1000), make(chan *work, 1000), diffMan, nil, nil}
	wrkr.logFile, _ = os.Create("log.txt")
	wrkr.logWriter = bufio.NewWriter(wrkr.logFile)
	return &wrkr
}

var depRe = regexp.MustCompile(`###   (.*):(.*):(.*):(.*)`)

func (w *Worker) Start() {
	w.startCheckExist()
	w.startClone()
	w.startEs()
}

func (w *Worker) startCheckExist() {
	work := func(worker int) {
		for {
			git := <-w.checkExistQueue
			log.Printf("[CHECK-WORKER (%d)] Starting work on %s\n", worker, git.AfterSha)
			if exists, err := es.CheckShaExists(w.index, git.Repository.FullName, git.AfterSha); err != nil {
				log.Printf("[CHECK-WORKER (%d)] Unable to check status of current sha: %s\n", worker, err.Error())
				continue
			} else if exists {
				log.Printf("[CHECK-WORKER (%d)] This sha already exists\n", worker)
				continue
			}
			log.Printf("[CHECK-WORKER (%d)] Adding %s to clone queue\n", worker, git.AfterSha)
			w.cloneQueue <- git
		}
	}
	for i := 1; i <= w.numWorkers; i++ {
		go work(i)
	}
}
func (w *Worker) startClone() {
	work := func(worker int) {
		for {
			git := <-w.cloneQueue
			log.Printf("[CLONE-WORKER (%d)] Starting work on %s\n", worker, git.AfterSha)
			var deps []es.Dependency
			var hashes []string
			dat, err := exec.Command(w.singleLocation, git.Repository.FullName, git.AfterSha).Output()
			if err != nil {
				log.Printf("[CLONE-WORKER (%d)] Unable to run against %s [%s]\n", worker, git.AfterSha, err.Error())
				w.logWriter.WriteString(u.Format("[CLONE-WORKER (%d)] Unable to run against %s [%s]\n[%s]\n", worker, git.AfterSha, err.Error(), string(dat)))
				w.logWriter.Flush()
				continue
			}
			{
				tmp := strings.Split(string(dat), "\n")[2:]
				deps = make([]es.Dependency, 0, len(tmp))
				for _, p := range tmp {
					if p == "" {
						continue
					}
					matches := depRe.FindStringSubmatch(p)
					deps = append(deps, es.Dependency{matches[1], matches[2], matches[4]})
				}
			}
			{
				hashes = make([]string, len(deps))
				for i, d := range deps {
					hash := d.GetHashSum()
					hashes[i] = hash
					exists, err := w.index.ItemExists("dependency", hash)
					if err != nil || !exists {
						go func(dep es.Dependency, h string) {
							resp, err := w.index.PostData("dependency", h, dep)
							if err != nil {
								log.Printf("[CLONE-WORKER (%d)] Unable to create dependency %s [%s]\n", worker, h, err.Error())
							} else if !resp.Created {
								log.Printf("[CLONE-WORKER (%d)] Unable to create dependency %s\n", worker, h)
							}
						}(d, hash)
					}
				}
				sort.Strings(hashes)
			}
			log.Printf("[CLONE-WORKER (%d)] Adding %s to es queue\n", worker, git.AfterSha)
			w.esQueue <- &work{git.Repository.FullName, git.Repository.Name, git.AfterSha, git.Ref, hashes, git.Real}
		}
	}
	for i := 1; i <= w.numWorkers; i++ {
		go work(i)
	}
}

func (w *Worker) startEs() {
	work := func() {
		for {
			workInfo := <-w.esQueue
			log.Println("[ES-WORKER] Starting work on", workInfo.sha)
			docName := strings.Replace(workInfo.fullName, "/", "_", -1)
			var exists bool
			var err error
			var project *es.Project
			var ref *es.Ref

			if exists, err = w.index.ItemExists("project", docName); err != nil {
				log.Println("[ES-WORKER] Error checking project exists:", err.Error())
				continue
			}
			if exists {
				project, err = es.GetProjectById(w.index, docName)
				if err != nil {
					log.Println("[ES-WORKER] Unable to retrieve project:", err.Error())
					continue
				}
			} else {
				project = es.NewProject(workInfo.fullName, workInfo.name)
			}
			for _, r := range project.Refs {
				if r.Name == workInfo.ref {
					ref = &r
					break
				}
			}
			if ref == nil {
				project.Refs = append(project.Refs, *es.NewRef(workInfo.ref))
				ref = &project.Refs[len(project.Refs)-1]
			}
			newEntry := es.ProjectEntry{Sha: workInfo.sha}
			if workInfo.reall {
				if len(ref.WebhookOrder) > 0 {
					testReferenceSha := ref.WebhookOrder[0]
					testReference := ref.MustGetEntry(testReferenceSha)
					if testReference.EntryReference != "" {
						testReferenceSha = testReference.EntryReference
						testReference = ref.MustGetEntry(testReferenceSha)
					}
					if reflect.DeepEqual(workInfo.hashes, testReference.Dependencies) {
						newEntry.EntryReference = testReferenceSha
					} else {
						newEntry.Dependencies = workInfo.hashes
					}

				}
				ref.WebhookOrder = append([]string{workInfo.sha}, ref.WebhookOrder...)
			} else {
				newEntry.Dependencies = workInfo.hashes
			}

			ref.Entries = append(ref.Entries, newEntry)

			if strings.HasPrefix(workInfo.ref, "refs/tags/") {
				tag := strings.Split(workInfo.ref, "/")[2]
				project.TagShas = append(project.TagShas, es.TagSha{Tag: tag, Sha: workInfo.sha})
			}

			indexProject := func(data func(string, string, interface{}) (*elasticsearch.IndexResponse, error), method string, checkCreate bool) bool {
				resp, err := data("project", docName, project)
				if err != nil {
					log.Println("[ES-WORKER] Unable to", method, "project:", err.Error())
					return true
				} else if !resp.Created && checkCreate {
					log.Println("[ES-WORKER] Project was not created")
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
			log.Println("[ES-WORKER] Finished work on", workInfo.fullName, workInfo.sha)
			if workInfo.reall {
				go func() {
					_, err := w.diffMan.webhookCompare(project.FullName, ref)
					if err != nil {
						log.Println("[ES-WORKER] Error creating diff:", err.Error())
					}
				}()
			}
		}
	}
	go work()
}

func (w *Worker) AddTask(git *s.GitWebhook) {
	w.checkExistQueue <- git
}
