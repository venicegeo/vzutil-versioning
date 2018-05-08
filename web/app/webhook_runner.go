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
	"strings"

	"github.com/venicegeo/pz-gocommon/elasticsearch"
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
	docName := strings.Replace(workInfo.fullName, "/", "_", -1)
	var exists bool
	var err error
	var repo *es.Repository
	var ref *es.Ref

	if exists, err = w.app.index.ItemExists("repository", docName); err != nil {
		log.Println("[ES-WORKER] Error checking repository exists:", err.Error())
		return
	}
	if exists {
		repo, _, err = es.GetRepositoryById(w.app.index, docName)
		if err != nil {
			log.Println("[ES-WORKER] Unable to retrieve repository:", err.Error())
			return
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
			return
		}
	} else { //PUT
		if indexRepository(w.app.index.PutData, "put", false) {
			return
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
