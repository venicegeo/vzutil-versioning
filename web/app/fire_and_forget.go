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

	s "github.com/venicegeo/vzutil-versioning/web/app/structs"
	"github.com/venicegeo/vzutil-versioning/web/es"
)

type FireAndForget struct {
	app *Application
}

func NewFireAndForget(app *Application) *FireAndForget {
	return &FireAndForget{app}
}

func (w *FireAndForget) FireRequest(request *SingleRunnerRequest) {
	go func(request *SingleRunnerRequest) {
		exists := make(chan bool, 1)
		ret := make(chan *RepositoryDependencyScan, 1)
		defer func() {
			close(exists)
			close(ret)
		}()
		w.app.wrkr.AddTask(request, exists, ret)
		e := <-exists
		if e {
			return
		}
		r := <-ret
		if r != nil {
			w.postScan(r)
		}
	}(request)
}

func (w *FireAndForget) FireGit(git *s.GitWebhook) {
	go func(git *s.GitWebhook) {
		fire := func(git *s.GitWebhook, repo *Repository) {
			ret := make(chan *RepositoryDependencyScan, 1)
			defer close(ret)
			request := &SingleRunnerRequest{
				repository: repo,
				sha:        git.AfterSha,
				ref:        git.Ref,
			}
			w.app.wrkr.AddTask(request, nil, ret)
			r := <-ret
			if r != nil {
				w.postScan(r)
			}
		}

		log.Println("[RECIEVED WEBHOOK]", git.Repository.FullName, git.AfterSha, git.Ref)
		if projects, err := w.app.rtrvr.GetAllProjectNamesUsingRepository(git.Repository.FullName); err != nil {
			log.Println("FAILED TO FIND PROJECTS USING REPOSITORY FOR WEBHOOK", git.AfterSha)
		} else {
			for _, p := range projects {
				go func(p string) {
					if repo, _, err := w.app.rtrvr.GetRepository(git.Repository.FullName, p); err != nil {
						log.Println("FAILED TO GET THE REPO INSTANCE UNDER", p)
					} else {
						go fire(git, repo)
					}
				}(p)
			}
		}
	}(git)

}

func (w *FireAndForget) postScan(scan *RepositoryDependencyScan) {
	log.Println("[ES-WORKER] Starting work on", scan.Sha, "for", scan.Project)
	var err error

	var testAgainstEntry *RepositoryDependencyScan
	//TODO difference between term and terms?
	boolq := es.NewBool().
		SetMust(es.NewBoolQ(
			es.NewTerm(Scan_FullnameField, scan.RepoFullname),
			es.NewTerms(Scan_RefsField, scan.Refs...),
			es.NewRange(Scan_TimestampField, "lt", scan.Timestamp)))
	result, err := w.app.index.SearchByJSON(RepositoryEntryType, map[string]interface{}{
		"query": map[string]interface{}{"bool": boolq},
		"sort": map[string]interface{}{
			Scan_TimestampField: "desc",
		},
		"size": 1,
	})
	if err == nil {
		if result.Hits.TotalHits == 1 {
			testAgainstEntr := new(RepositoryDependencyScan)
			if err = json.Unmarshal(*result.Hits.Hits[0].Source, testAgainstEntr); err == nil {
				testAgainstEntry = testAgainstEntr
			}
		}
	}

	resp, err := w.app.index.PostData(RepositoryEntryType, scan.Sha+"-"+scan.Project, scan)
	if err != nil {
		log.Printf("[ES-WORKER] Unable to create entry %s: %s\n", scan.Sha, err.Error())
		return
	} else if !resp.Created {
		log.Printf("[ES-WORKER] Unable to create entry %s. No error\n", scan.Sha)
		return
	}

	log.Println("[ES-WORKER] Finished work on", scan.RepoFullname, scan.Sha)
	go func(fullName string, testAgainstEntry, entry *RepositoryDependencyScan) {
		if testAgainstEntry == nil {
			return
		}
		_, err := w.app.diffMan.webhookCompare(testAgainstEntry, entry)
		if err != nil {
			log.Println("[ES-WORKER] Error creating diff:", err.Error())
		}
	}(scan.RepoFullname, testAgainstEntry, scan)
}
