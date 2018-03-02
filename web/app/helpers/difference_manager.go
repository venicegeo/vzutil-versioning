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
	"encoding/json"
	"sort"
	"time"

	"github.com/venicegeo/pz-gocommon/elasticsearch"
	"github.com/venicegeo/vzutil-versioning/web/es"
	u "github.com/venicegeo/vzutil-versioning/web/util"
	t "github.com/venicegeo/vzutil-versioning/web/util/table"
)

type DifferenceManager struct {
	index *elasticsearch.Index

	CurrentDisplay string
}

func NewDifferenceManager(index *elasticsearch.Index) *DifferenceManager {
	return &DifferenceManager{index, ""}
}

type Difference struct {
	Id       string   `json:"id"`
	FullName string   `json:"full_name"`
	OldSha   string   `json:"old_sha"`
	NewSha   string   `json:"new_sha"`
	Removed  []string `json:"removed"`
	Added    []string `json:"added"`
	NanoTime int64    `json:"time"`
}

func (d *Difference) SimpleString() string {
	return d.FullName + " " + time.Unix(0, d.NanoTime).Format(time.RFC3339)
}

type diffSort []Difference

func (d diffSort) Len() int {
	return len(d)
}
func (d diffSort) Swap(i, j int) {
	d[i], d[j] = d[j], d[i]
}
func (d diffSort) Less(i, j int) bool {
	return d[i].NanoTime > d[j].NanoTime
}

func (dm *DifferenceManager) GenerateReport(d *Difference) string {
	getDep := func(dep string) string {
		if resp, err := dm.index.GetByID("dependency", dep); err != nil || !resp.Found {
			name := u.Format("Cound not find [%s]", dep)
			tmp := es.Dependency{name, "", ""}
			return u.Format("%s:%s:%s", tmp.Name, tmp.Version, tmp.Language)
		} else {
			var depen es.Dependency
			if err = json.Unmarshal([]byte(*resp.Source), &depen); err != nil {
				tmp := es.Dependency{u.Format("Error getting [%s]: [%s]", dep, err.Error()), "", ""}
				return u.Format("%s:%s:%s", tmp.Name, tmp.Version, tmp.Language)
			} else {
				return u.Format("%s:%s:%s", depen.Name, depen.Version, depen.Language)
			}
		}
	}
	height := len(d.Removed)
	if height < len(d.Added) {
		height = len(d.Added)
	}
	table := t.NewTable(2, height+1)
	table.Fill("Removed")
	table.Fill("Added")
	for i := 0; i < height; i++ {
		if i < len(d.Removed) {
			table.Fill(getDep(d.Removed[i]))
		} else {
			table.Fill("")
		}
		if i < len(d.Added) {
			table.Fill(getDep(d.Added[i]))
		} else {
			table.Fill("")
		}
	}
	return u.Format("Project %s from\n%s -> %s\n%s", d.FullName, d.OldSha, d.NewSha, table.Format().NoRowBorders().SpaceAllColumns().String())
}

func (d *DifferenceManager) AllDiffs(size int) (*[]Difference, error) {
	resp, err := es.MatchAllSize(d.index, "difference", size)
	if err != nil {
		return nil, err
	}
	hits := resp.GetHits()
	diffs := make([]Difference, len(*hits))
	for i, hit := range *hits {
		var diff Difference
		if err = json.Unmarshal([]byte(*hit.Source), &diff); err != nil {
			return nil, err
		}
		diffs[i] = diff
	}
	sort.Sort(diffSort(diffs))
	return &diffs, nil
}

func (d *DifferenceManager) DiffList(size int) ([]string, error) {
	diffs, err := d.AllDiffs(size)
	if err != nil {
		return nil, err
	}
	res := make([]string, len(*diffs))
	for i, diff := range *diffs {
		res[i] = diff.SimpleString()
	}
	return res, nil
}

func (d *DifferenceManager) ShaCompare(fullName, oldSha, newSha string) (*Difference, error) {
	t := time.Now().UnixNano()
	project, err := es.GetProjectById(d.index, fullName)
	if err != nil {
		return nil, err
	}
	_, oldEntry, ok := project.GetEntry(oldSha)
	if !ok {
		return nil, u.Error("Could not get old entry")
	}
	_, newEntry, ok := project.GetEntry(newSha)
	if !ok {
		return nil, u.Error("Could not get new entry")
	}
	return d.diffCompareWrk(fullName, oldEntry, newEntry, oldSha, newSha, t)
}

func (d *DifferenceManager) webhookCompare(fullName string, ref *es.Ref) (*Difference, error) {
	if len(ref.WebhookOrder) < 2 {
		return nil, nil
	}
	t := time.Now().UnixNano()
	oldSha := ref.WebhookOrder[1]
	newSha := ref.WebhookOrder[0]

	newEntry, ok := ref.GetEntry(newSha)
	if !ok {
		return nil, u.Error("Could not get new entry")
	} else if newEntry.EntryReference == oldSha {
		return nil, nil
	}
	oldEntry, ok := ref.GetEntry(oldSha)
	if !ok {
		return nil, u.Error("Could not get old entry")
	}
	return d.diffCompareWrk(fullName, oldEntry, newEntry, oldSha, newSha, t)
}

func (d *DifferenceManager) diffCompareWrk(fullName string, oldEntry, newEntry es.ProjectEntry, oldSha, newSha string, t int64) (*Difference, error) {
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
	id := u.Hash(u.Format("%s%d", fullName, t))
	diff := Difference{id, fullName, oldSha, newSha, removed, added, t}
	resp, err := d.index.PostData("difference", id, diff)
	if err != nil {
		return nil, err
	}
	if !resp.Created {
		return nil, u.Error("Diff was not created")
	}
	return &diff, nil
}

func (d *DifferenceManager) Delete(id string) {
	es.DeleteAndWait(d.index, "difference", id)
}

func strscont(sl []string, s string) bool {
	for _, ss := range sl {
		if ss == s {
			return true
		}
	}
	return false
}
