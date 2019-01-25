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
	"sync"

	h "github.com/venicegeo/vzutil-versioning/common/history"
	"github.com/venicegeo/vzutil-versioning/web/es/types"
	u "github.com/venicegeo/vzutil-versioning/web/util"
)

type Worker struct {
	app *Application

	snglRnnr        *SingleRunner
	numWorkers      int
	mux             *sync.Mutex
	checkWorking    []bool
	checkExistQueue chan *existsWork
	cloneWorking    []bool
	cloneQueue      chan *scanWork
}

type existsWork struct {
	request   *SingleRunnerRequest
	exists    chan *types.Scan
	singleRet chan *types.Scan
}
type scanWork struct {
	request   *SingleRunnerRequest
	singleRet chan *types.Scan
}

func NewWorker(app *Application, numWorkers int) *Worker {
	wrkr := Worker{app, NewSingleRunner(app), numWorkers, &sync.Mutex{}, make([]bool, numWorkers), make(chan *existsWork, 1000), make([]bool, numWorkers), make(chan *scanWork, 1000)}
	return &wrkr
}

func (w *Worker) JobsInSystem() int {
	w.mux.Lock()
	res := len(w.checkExistQueue) + len(w.cloneQueue)
	for _, b := range w.checkWorking {
		if b {
			res++
		}
	}
	for _, b := range w.cloneWorking {
		if b {
			res++
		}
	}
	w.mux.Unlock()
	return res
}

func (w *Worker) Start() {
	w.startCheckExist()
	w.startClone()
}

func (w *Worker) startCheckExist() {
	work := func(worker int) {
		for {
			work := <-w.checkExistQueue
			w.mux.Lock()
			w.checkWorking[worker] = true
			w.mux.Unlock()
			log.Printf("[CHECK-WORKER (%d)] Starting work on %s\n", worker, work.request.sha)

			item, err := w.app.index.GetByID(RepositoryEntryType, work.request.sha+"-"+work.request.repository.ProjectId)
			if err != nil || !item.Found {
				log.Printf("[CHECK-WORKER (%d)] Unable to check status of current sha: %s. Continuing\n", worker, err.Error())
			}
			if item.Found {
				repEntry := new(types.Scan)
				if err = json.Unmarshal(*item.Source, repEntry); err != nil {
					log.Printf("[CHECK-WORKER (%d)] Unable to unmarshal sha: %s\nReason: %s\n", worker, work.request.sha, err.Error())
					work.exists <- nil
					work.singleRet <- nil
					continue
				}
				work.exists <- repEntry
				log.Printf("[CHECK-WORKER (%d)] This sha already exists\n", worker)
				continue
			}
			work.exists <- nil
			log.Printf("[CHECK-WORKER (%d)] Adding %s to clone queue\n", worker, work.request.sha)
			w.cloneQueue <- &scanWork{work.request, work.singleRet}
			w.mux.Lock()
			w.checkWorking[worker] = false
			w.mux.Unlock()
		}
	}
	for i := 0; i < w.numWorkers; i++ {
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
			w.mux.Lock()
			w.cloneWorking[worker] = true
			w.mux.Unlock()

			singleRet := w.snglRnnr.RunAgainstSingle(u.Format("[CLONE-WORKER (%d)] ", worker), toPrint, work.request)
			done <- true
			work.singleRet <- singleRet
			w.mux.Lock()
			w.cloneWorking[worker] = false
			w.mux.Unlock()
		}
	}
	for i := 0; i < w.numWorkers; i++ {
		go work(i)
	}
}

func (w *Worker) AddTask(request *SingleRunnerRequest, exists chan *types.Scan, singleRet chan *types.Scan) {
	if exists != nil {
		w.checkExistQueue <- &existsWork{request, exists, singleRet}
	} else {
		w.cloneQueue <- &scanWork{request, singleRet}
	}
}

func (w *Worker) History(repo string) (h.HistoryTree, error) {
	return w.snglRnnr.History(repo)
}
