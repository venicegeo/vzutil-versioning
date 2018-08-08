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

	c "github.com/venicegeo/vzutil-versioning/common"
	s "github.com/venicegeo/vzutil-versioning/web/app/structs"
	"github.com/venicegeo/vzutil-versioning/web/es"
	u "github.com/venicegeo/vzutil-versioning/web/util"
)

type WebhookRunner struct {
	app *Application
}

func NewWebhookRunner(app *Application) *WebhookRunner {
	return &WebhookRunner{app}
}

func (w *WebhookRunner) RunAgainstWeb(git *s.GitWebhook) {
	go func(git *s.GitWebhook) {
		exists := make(chan bool, 1)
		ret := make(chan *SingleResult, 1)
		w.app.wrkr.AddTask(git, exists, ret)
		e := <-exists
		if e {
			return
		}
		r := <-ret
		if r != nil {
			w.es(r)
		}
	}(git)
}

func (w *WebhookRunner) es(workInfo *c.DependencyScan) {
	log.Println("[ES-WORKER] Starting work on", workInfo.sha)
	var err error

	var testAgainstEntry *c.DependencyScan
	result, err := w.app.index.SearchByJSON("repository_entry", u.Format(`
{
	"query":{
		"bool":{
			"must":[
				{
					"term":{
						"repo_fullname":"%s"
					}
				},
				{
					"term":{
							"ref_names":"[%s]"
					}
				},
				{
					"range":{
						"timestamp":{ "lt":%d }
					}
				}
			]
		}
	},
	"sort":{
		"timestamp":"desc"
	},
	"size":1
}`, workInfo.fullName, workInfo.ref, workInfo.timestamp))
	if err == nil {
		if result.NumHits() == 1 {
			var testAgainstEntr es.RepositoryEntry
			if err = json.Unmarshal(*result.GetHit(0).Source, &testAgainstEntr); err == nil {
				testAgainstEntry = &testAgainstEntr
			}
		}
	}

	resp, err := w.app.index.PostData("repository_entry", workInfo.sha, entry)
	if err != nil {
		log.Printf("[ES-WORKER] Unable to create entry %s: %s\n", workInfo.sha, err.Error())
		return
	} else if !resp.Created {
		log.Printf("[ES-WORKER] Unable to create entry %s\n", workInfo.sha)
		return
	}

	log.Println("[ES-WORKER] Finished work on", workInfo.fullName, workInfo.sha)
	go func(fullName string, testAgainstEntry *es.RepositoryEntry, entry es.RepositoryEntry) {
		if testAgainstEntry == nil {
			return
		}
		_, err := w.app.diffMan.webhookCompare(*testAgainstEntry, entry)
		if err != nil {
			log.Println("[ES-WORKER] Error creating diff:", err.Error())
		}
	}(workInfo.fullName, testAgainstEntry, entry)
}
