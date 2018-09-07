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

func (ff *FireAndForget) FireRequest(request *SingleRunnerRequest) {
	go func(request *SingleRunnerRequest) {
		exists := make(chan *RepositoryDependencyScan, 1)
		ret := make(chan *RepositoryDependencyScan, 1)
		defer func() {
			close(exists)
			close(ret)
		}()
		ff.app.wrkr.AddTask(request, exists, ret)
		e := <-exists
		if e != nil {
			ff.tryUpdateScan(request.ref, e)
		} else {
			r := <-ret
			if r != nil {
				ff.postScan(r)
			}
		}
	}(request)
}

func (ff *FireAndForget) FireGit(git *s.GitWebhook) {
	go func(git *s.GitWebhook) {
		fire := func(git *s.GitWebhook, repo *Repository) {
			ret := make(chan *RepositoryDependencyScan, 1)
			defer close(ret)
			request := &SingleRunnerRequest{
				repository: repo,
				sha:        git.AfterSha,
				ref:        git.Ref,
			}
			ff.app.wrkr.AddTask(request, nil, ret)
			r := <-ret
			if r != nil {
				ff.postScan(r)
			}
		}

		log.Println("[RECIEVED WEBHOOK]", git.Repository.FullName, git.AfterSha, git.Ref)
		if projects, err := ff.app.rtrvr.GetAllProjectNamesUsingRepository(git.Repository.FullName); err != nil {
			log.Println("FAILED TO FIND PROJECTS USING REPOSITORY FOR WEBHOOK", git.AfterSha)
		} else {
			for _, p := range projects {
				go func(p string) {
					if repo, _, err := ff.app.rtrvr.GetRepository(git.Repository.FullName, p); err != nil {
						log.Println("FAILED TO GET THE REPO INSTANCE UNDER", p)
					} else {
						go fire(git, repo)
					}
				}(p)
			}
		}
	}(git)

}

func (ff *FireAndForget) tryUpdateScan(ref string, scan *RepositoryDependencyScan) {
	contains := false
	for _, r := range scan.Refs {
		if r == ref {
			contains = true
			break
		}
	}
	if contains {
		return
	}
	scan.Refs = append(scan.Refs, ref)

	_, err := ff.app.index.PostData(RepositoryEntryType, scan.Sha+"-"+scan.ProjectId, scan)
	if err != nil {
		log.Printf("[ES-WORKER] Unable to update entry %s: %s\n", scan.Sha, err.Error())
	} else {
		log.Println("[ES-WORKER] Updated", scan.Sha, "for", scan.ProjectId, "with ref", ref)
	}
}

func (ff *FireAndForget) postScan(scan *RepositoryDependencyScan) {
	log.Println("[ES-WORKER] Starting work on", scan.Sha, "for", scan.ProjectId)
	var err error

	testAgainstEntries := make(map[string]*RepositoryDependencyScan, len(scan.Refs))
	for _, ref := range scan.Refs {
		boolq := es.NewBool().
			SetMust(es.NewBoolQ(
				es.NewTerm(Scan_FullnameField, scan.RepoFullname),
				es.NewTerm(Scan_RefsField, ref),
				es.NewTerm(Scan_ProjectIdField, scan.ProjectId),
				es.NewRange(Scan_TimestampField, "lt", scan.Timestamp)))
		q := map[string]interface{}{
			"query": map[string]interface{}{"bool": boolq},
			"sort": map[string]interface{}{
				Scan_TimestampField: "desc",
			},
			"size": 1,
		}
		result, err := ff.app.index.SearchByJSON(RepositoryEntryType, q)
		if err == nil && result.Hits.TotalHits == 1 {
			entry := new(RepositoryDependencyScan)
			if err = json.Unmarshal(*result.Hits.Hits[0].Source, entry); err == nil {
				testAgainstEntries[ref] = entry
			}
		}
	}

	resp, err := ff.app.index.PostData(RepositoryEntryType, scan.Sha+"-"+scan.ProjectId, scan)
	if err != nil {
		log.Printf("[ES-WORKER] Unable to create entry %s: %s\n", scan.Sha, err.Error())
		return
	} else if !resp.Created {
		log.Printf("[ES-WORKER] Unable to create entry %s. No error\n", scan.Sha)
		return
	}

	log.Println("[ES-WORKER] Finished work on", scan.RepoFullname, scan.Sha)
	for ref, old := range testAgainstEntries {
		go ff.runDiff(scan.RepoFullname, scan.ProjectId, ref, old, scan)
	}
}

func (w *FireAndForget) runDiff(repoName, projectName, ref string, oldEntry, newEntry *RepositoryDependencyScan) {
	if _, err := w.app.diffMan.webhookCompare(repoName, projectName, ref, oldEntry, newEntry); err != nil {
		log.Println("[ES-WORKER] Error creating diff:", err.Error())
	}
}
