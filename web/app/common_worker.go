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
	"regexp"

	s "github.com/venicegeo/vzutil-versioning/web/app/structs"
	"github.com/venicegeo/vzutil-versioning/web/es"
	u "github.com/venicegeo/vzutil-versioning/web/util"
)

type Worker struct {
	app *Application

	snglRnnr        *SingleRunner
	numWorkers      int
	checkExistQueue chan *workS
	cloneQueue      chan *workS
}

type workS struct {
	gitInfo   *s.GitWebhook
	exists    chan bool
	singleRet chan *SingleResult
}

type SingleResult struct {
	fullName  string
	name      string
	sha       string
	ref       string
	Deps      []es.Dependency
	hashes    []string
	timestamp int64
}

func NewWorker(app *Application, numWorkers int) *Worker {
	wrkr := Worker{app, NewSingleRunner(app), numWorkers, make(chan *workS, 1000), make(chan *workS, 1000)}
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
			log.Printf("[CHECK-WORKER (%d)] Starting work on %s\n", worker, work.gitInfo.AfterSha)
			if exists, err := es.CheckShaExists(w.app.index, work.gitInfo.Repository.FullName, work.gitInfo.AfterSha); err != nil {
				log.Printf("[CHECK-WORKER (%d)] Unable to check status of current sha: %s\n", worker, err.Error())
				continue
			} else if exists {
				work.exists <- true
				log.Printf("[CHECK-WORKER (%d)] This sha already exists\n", worker)
				continue
			}
			work.exists <- false
			log.Printf("[CHECK-WORKER (%d)] Adding %s to clone queue\n", worker, work.gitInfo.AfterSha)
			w.cloneQueue <- work
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
			singleRet := w.snglRnnr.RunAgainstSingle(u.Format("[CLONE-WORKER (%d)] ", worker), toPrint, work.gitInfo)
			done <- true
			work.singleRet <- singleRet
		}
	}
	for i := 1; i <= w.numWorkers; i++ {
		go work(i)
	}
}

func (w *Worker) AddTask(git *s.GitWebhook, exists chan bool, singleRet chan *SingleResult) {
	w.checkExistQueue <- &workS{git, exists, singleRet}
}
