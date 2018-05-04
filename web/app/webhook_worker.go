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

package app

import (
	"log"
	"reflect"
	"regexp"
	"strings"

	"github.com/venicegeo/pz-gocommon/elasticsearch"
	s "github.com/venicegeo/vzutil-versioning/web/app/structs"
	"github.com/venicegeo/vzutil-versioning/web/es"
	u "github.com/venicegeo/vzutil-versioning/web/util"
)

type Worker struct {
	app *Application

	numWorkers      int
	checkExistQueue chan *s.GitWebhook
	cloneQueue      chan *s.GitWebhook
	esQueue         chan *SingleResult
}

type SingleResult struct {
	fullName string
	name     string
	sha      string
	ref      string
	Deps     []es.Dependency
	hashes   []string
}

func NewWorker(app *Application, numWorkers int) *Worker {
	wrkr := Worker{app, numWorkers, make(chan *s.GitWebhook, 1000), make(chan *s.GitWebhook, 1000), make(chan *SingleResult, 1000)}
	return &wrkr
}

var depRe = regexp.MustCompile(`(.*):(.*):(.*)`)

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
			if exists, err := es.CheckShaExists(w.app.index, git.Repository.FullName, git.AfterSha); err != nil {
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
			done := make(chan bool, 1)
			toPrint := make(chan string, 6)
			go func() {
				for {
					select {
					case x := <-toPrint:
						log.Println(x)
					case <-done:
						return
					}
				}
			}()
			w.app.snglRnnr.RunAgainstSingleChan(u.Format("[CLONE-WORKER (%d)] ", worker), toPrint, w.cloneQueue, w.esQueue)
			done <- true
		}
	}
	for i := 1; i <= w.numWorkers; i++ {
		go work(i)
	}
}

//[CLONE-WORKER (%d)]
func (w *Worker) startEs() {
	work := func() {
		for {
			workInfo := <-w.esQueue
			if workInfo == nil {
				continue
			}
			log.Println("[ES-WORKER] Starting work on", workInfo.sha)
			docName := strings.Replace(workInfo.fullName, "/", "_", -1)
			var exists bool
			var err error
			var repo *es.Repository
			var ref *es.Ref

			if exists, err = w.app.index.ItemExists("repository", docName); err != nil {
				log.Println("[ES-WORKER] Error checking repository exists:", err.Error())
				continue
			}
			if exists {
				repo, _, err = es.GetRepositoryById(w.app.index, docName)
				if err != nil {
					log.Println("[ES-WORKER] Unable to retrieve repository:", err.Error())
					continue
				}
			} else {
				repo = es.NewRepository(workInfo.fullName, workInfo.name)
			}
			for _, r := range repo.Refs {
				if r.Name == workInfo.ref {
					ref = r
					break
				}
			}
			if ref == nil {
				repo.Refs = append(repo.Refs, es.NewRef(workInfo.ref))
				ref = repo.Refs[len(repo.Refs)-1]
			}
			newEntry := es.RepositoryEntry{Sha: workInfo.sha}
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
			} else {
				newEntry.Dependencies = workInfo.hashes
			}
			ref.WebhookOrder = append([]string{workInfo.sha}, ref.WebhookOrder...)

			ref.Entries = append(ref.Entries, newEntry)

			if strings.HasPrefix(workInfo.ref, "refs/tags/") {
				tag := strings.Split(workInfo.ref, "/")[2]
				repo.TagShas = append(repo.TagShas, es.TagSha{Tag: tag, Sha: workInfo.sha})
			}

			indexRepository := func(data func(string, string, interface{}) (*elasticsearch.IndexResponse, error), method string, checkCreate bool) bool {
				resp, err := data("repository", docName, repo)
				if err != nil {
					log.Println("[ES-WORKER] Unable to", method, "repository:", err.Error())
					return true
				} else if !resp.Created && checkCreate {
					log.Println("[ES-WORKER] Repository was not created")
					return true
				}
				return false
			}
			if !exists { //POST
				if indexRepository(w.app.index.PostData, "post", true) {
					continue
				}
			} else { //PUT
				if indexRepository(w.app.index.PutData, "put", false) {
					continue
				}
			}
			log.Println("[ES-WORKER] Finished work on", workInfo.fullName, workInfo.sha)
			go func() {
				_, err := w.app.diffMan.webhookCompare(repo.FullName, ref)
				if err != nil {
					log.Println("[ES-WORKER] Error creating diff:", err.Error())
				}
			}()
		}
	}
	go work()
}

func (w *Worker) AddTask(git *s.GitWebhook) {
	w.checkExistQueue <- git
}
