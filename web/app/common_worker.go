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
	"encoding/json"
	"log"
	"regexp"

	c "github.com/venicegeo/vzutil-versioning/common"
	s "github.com/venicegeo/vzutil-versioning/web/app/structs"
	u "github.com/venicegeo/vzutil-versioning/web/util"
)

type Worker struct {
	app *Application

	snglRnnr        *SingleRunner
	numWorkers      int
	checkExistQueue chan *existsWork
	cloneQueue      chan *scanWork
}

type existsWork struct {
	request   *SingleRunnerRequest
	exists    chan bool
	singleRet chan *c.DependencyScan
}
type scanWork struct {
	request   *SingleRunnerRequest
	singleRet chan *c.DependencyScan
}

func NewWorker(app *Application, numWorkers int) *Worker {
	wrkr := Worker{app, NewSingleRunner(app), numWorkers, make(chan *existsWork, 1000), make(chan *scanWork, 1000)}
	return &wrkr
}

var depRe = regexp.MustCompile(`(.*):(.*):(.*)`)

func (w *Worker) Start() {
	w.startCheckExist()
	w.startClone()
}

func (w *Worker) startCheckExist() {
	work := func(worker int) {
		for {
			work := <-w.checkExistQueue
			log.Printf("[CHECK-WORKER (%d)] Starting work on %s\n", worker, work.request.Sha)

			item, err := w.app.index.GetByID("repository_entry", work.request.Sha)
			if err != nil {
				log.Printf("[CHECK-WORKER (%d)] Unable to check status of current sha: %s. Continuing\n", worker, err.Error())
			}
			if item.Found {
				var repEntry c.DependencyScan
				if err = json.Unmarshal(*item.Source, &repEntry); err != nil {
					log.Printf("[CHECK-WORKER (%d)] Unable to unmarshal sha: %s\nReason: %s\n", worker, work.request.Sha, err.Error())
					work.exists <- false
					work.singleRet <- nil
					continue
				}
				work.exists <- true
				go func(work *existsWork) {
					log.Printf("[CHECK-WORKER (%d)] This sha already exists\n", worker)
					refFound := false
					if work.request.Ref == "" {
						return
					}
					for _, name := range repEntry.Refs {
						if name == work.request.Ref {
							refFound = true
							break
						}
					}
					if !refFound {
						log.Printf("[CHECK-WORKER (%d)] Adding ref [%s] to sha [%s]\n", worker, work.request.Ref, work.request.Sha)
						repEntry.Refs = append(repEntry.Refs, work.request.Ref)
						w.app.index.PostData("repository_entry", work.request.Sha, repEntry)
					}
				}(work)
				continue
			}
			work.exists <- false
			log.Printf("[CHECK-WORKER (%d)] Adding %s to clone queue\n", worker, work.request.Sha)
			w.cloneQueue <- &scanWork{work.request, work.singleRet}
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
			work := <-w.cloneQueue

			singleRet := w.snglRnnr.RunAgainstSingle(u.Format("[CLONE-WORKER (%d)] ", worker), toPrint, work.request)
			done <- true
			work.singleRet <- singleRet
		}
	}
	for i := 1; i <= w.numWorkers; i++ {
		go work(i)
	}
}

func (w *Worker) AddTaskGit(git *s.GitWebhook, singleRet chan *c.DependencyScan) {
	w.cloneQueue <- &scanWork{&SingleRunnerRequest{git.Repository.FullName, git.AfterSha, git.Ref, ""}, singleRet}
}
func (w *Worker) AddTaskRequest(request *SingleRunnerRequest, exists chan bool, singleRet chan *c.DependencyScan) {
	w.checkExistQueue <- &existsWork{request, exists, singleRet}
}
