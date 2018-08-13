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
	"strings"

	c "github.com/venicegeo/vzutil-versioning/common"
	s "github.com/venicegeo/vzutil-versioning/web/app/structs"
	u "github.com/venicegeo/vzutil-versioning/web/util"
)

type CompleteRunner struct {
	app *Application
}

func NewCompleteRunner(app *Application) *CompleteRunner {
	return &CompleteRunner{app}
}

func (w *CompleteRunner) RunAgainstRequest(request *SingleRunnerRequest) {
	go func(request *SingleRunnerRequest) {
		exists := make(chan bool, 1)
		ret := make(chan *c.DependencyScan, 1)
		defer func() {
			close(exists)
			close(ret)
		}()
		w.app.wrkr.AddTaskRequest(request, exists, ret)
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

func (w *CompleteRunner) RunAgainstGit(git *s.GitWebhook, requester string, files []string) {
	go func(git *s.GitWebhook) {
		ret := make(chan *c.DependencyScan, 1)
		defer close(ret)
		w.app.wrkr.AddTaskGit(git, requester, files, ret)
		r := <-ret
		if r != nil {
			w.postScan(r)
		}
	}(git)
}

func (w *CompleteRunner) postScan(scan *c.DependencyScan) {
	log.Println("[ES-WORKER] Starting work on", scan.Sha, "for", scan.RequesterName)
	var err error

	var testAgainstEntry *c.DependencyScan
	result, err := w.app.index.SearchByJSON("repository_entry", u.Format(`
{
	"query":{
		"bool":{
			"must":[
				{
					"term":{
						"%s":"%s"
					}
				},
				{
					"term":{
							"%s": [%s]
					}
				},
				{
					"range":{
						"%s":{ "lt":%d }
					}
				}
			]
		}
	},
	"sort":{
		"%s":"desc"
	},
	"size":1
}`, c.FullNameField, scan.Fullname, c.RefsField, strings.TrimSuffix(strings.TrimPrefix(u.Format("%#v", scan.Refs), `[]string{`), `}`), c.TimestampField, scan.Timestamp, c.TimestampField))
	if err == nil {
		if result.Hits.TotalHits == 1 {
			var testAgainstEntr c.DependencyScan
			if err = json.Unmarshal(*result.Hits.Hits[0].Source, &testAgainstEntr); err == nil {
				testAgainstEntry = &testAgainstEntr
			}
		}
	}

	resp, err := w.app.index.PostData("repository_entry", scan.Sha+"-"+scan.RequesterName, scan)
	if err != nil {
		log.Printf("[ES-WORKER] Unable to create entry %s: %s\n", scan.Sha, err.Error())
		return
	} else if !resp.Created {
		log.Printf("[ES-WORKER] Unable to create entry %s. No error\n", scan.Sha)
		return
	}

	log.Println("[ES-WORKER] Finished work on", scan.Fullname, scan.Sha)
	go func(fullName string, testAgainstEntry, entry *c.DependencyScan) {
		if testAgainstEntry == nil {
			return
		}
		_, err := w.app.diffMan.webhookCompare(testAgainstEntry, entry)
		if err != nil {
			log.Println("[ES-WORKER] Error creating diff:", err.Error())
		}
	}(scan.Fullname, testAgainstEntry, scan)
}
