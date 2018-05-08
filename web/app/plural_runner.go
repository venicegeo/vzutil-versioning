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

	com "github.com/venicegeo/vzutil-versioning/common"
	u "github.com/venicegeo/vzutil-versioning/web/util"
)

type PluralRunner struct {
	app *Application
}

func NewPluralRunner(app *Application) *PluralRunner {
	return &PluralRunner{app}
}

func (pr *PluralRunner) RunAgainstPluralStr(repos, checkouts []string) (string, error) {
	ret, err := pr.RunAgainstPlural(repos, checkouts)
	if err != nil {
		return "", err
	}
	dat, err := json.MarshalIndent(ret, " ", "   ")
	return string(dat), nil
}
func (pr *PluralRunner) RunAgainstPlural(repos, checkouts []string) (*com.RepositoriesDependencies, error) {
	if len(repos) != len(checkouts) {
		return nil, u.Error("Inputs not the same length")
	}
	res := com.RepositoriesDependencies{}
	wg := sync.WaitGroup{}
	mux := sync.Mutex{}
	wg.Add(len(repos))
	work := func(i int) {
		deps, err := pr.app.rtrvr.DepsByShaNameGen(repos[i], checkouts[i])
		if err != nil {
			if err != nil {
				log.Println("Error running in plural:", err.Error())
			}
			return
		}
		repdep := com.RepositoryDependencies{
			Name: repos[i],
			Sha:  checkouts[i],
			Deps: nil,
		}
		sdeps := make([]string, len(deps), len(deps))
		for j, d := range deps {
			sdeps[j] = d.String()
		}
		repdep.Deps = sdeps
		mux.Lock()
		res[repos[i]] = repdep
		mux.Unlock()
		wg.Done()
	}
	for i := 0; i < len(repos); i++ {
		go work(i)
	}
	wg.Wait()
	return &res, nil
}
