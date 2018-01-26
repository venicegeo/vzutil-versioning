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

package helpers

import (
	"errors"
	"time"

	"github.com/venicegeo/pz-gocommon/elasticsearch"
	"github.com/venicegeo/vzutil-versioning/web/es"
)

type DifferenceManager struct {
	index *elasticsearch.Index
}

func NewDifferenceManager(index *elasticsearch.Index) *DifferenceManager {
	return &DifferenceManager{index}
}

type Difference struct {
	FullName string   `json:"full_name"`
	OldSha   string   `json:"old_sha"`
	NewSha   string   `json:"new_sha"`
	Removed  []string `json:"removed"`
	Added    []string `json:"added"`
	Time     int64    `json:"time"`
}

func (d *DifferenceManager) webhookCompare(project *es.Project) (*Difference, error) {
	if len(project.WebhookOrder) < 2 {
		return nil, nil
	}
	t := time.Now().Unix()
	oldSha := project.WebhookOrder[1]
	newSha := project.WebhookOrder[0]

	entries, err := project.GetEntries()
	if err != nil {
		return nil, err
	}
	newEntry, ok := (*entries)[newSha]
	if !ok {
		return nil, errors.New("Could not get new entry")
	} else if newEntry.EntryReference == oldSha {
		return nil, nil
	}
	oldEntry, ok := (*entries)[oldSha]
	if !ok {
		return nil, errors.New("Could not get old entry")
	}

	added := []string{}
	removed := []string{}

	done := make(chan bool, 2)

	go func() {
		for _, newDep := range newEntry.Dependencies {
			if !strscont(oldEntry.Dependencies, newDep) {
				added = append(added, newDep)
			}
		}
		done <- true
	}()
	go func() {
		for _, oldDep := range oldEntry.Dependencies {
			if !strscont(newEntry.Dependencies, oldDep) {
				removed = append(removed, oldDep)
			}
		}
		done <- true
	}()
	for i := 0; i < 2; i++ {
		<-done
	}
	if len(added) == 0 && len(removed) == 0 {
		return nil, nil
	}
	diff := Difference{project.FullName, oldSha, newSha, removed, added, t}
	resp, err := d.index.PostData("difference", "", diff)
	if err != nil {
		return nil, err
	}
	if !resp.Created {
		return nil, errors.New("Diff was not created")
	}
	return &diff, nil

}

func strscont(sl []string, s string) bool {
	for _, ss := range sl {
		if ss == s {
			return true
		}
	}
	return false
}
