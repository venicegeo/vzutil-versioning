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

	s "github.com/venicegeo/vzutil-versioning/web/app/structs"
	"github.com/venicegeo/vzutil-versioning/web/es"
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

func (w *WebhookRunner) es(workInfo *SingleResult) {
	log.Println("[ES-WORKER] Starting work on", workInfo.sha)
	//	docName := strings.Replace(workInfo.fullName, "/", "_", -1)
	//	var exists bool
	var err error

	entry := es.RepositoryEntry{
		RepositoryFullName: workInfo.fullName,
		RepositoryName:     workInfo.name,
		RefName:            workInfo.ref,
		Sha:                workInfo.sha,
		Timestamp:          workInfo.timestamp,
		Dependencies:       workInfo.hashes,
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
	//	go func() {
	//		_, err := w.app.diffMan.webhookCompare(repo.FullName, ref)
	//		if err != nil {
	//			log.Println("[ES-WORKER] Error creating diff:", err.Error())
	//		}
	//	}()
}
