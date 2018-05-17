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
	"sort"
	"time"

	t "github.com/venicegeo/vzutil-versioning/common/table"
	"github.com/venicegeo/vzutil-versioning/web/es"
	u "github.com/venicegeo/vzutil-versioning/web/util"
)

type DifferenceManager struct {
	app *Application

	CurrentDisplay string
}

func NewDifferenceManager(app *Application) *DifferenceManager {
	return &DifferenceManager{app, ""}
}

type Difference struct {
	Id       string   `json:"id"`
	FullName string   `json:"full_name"`
	RefData  string   `json:"ref"`
	OldSha   string   `json:"old_sha"`
	NewSha   string   `json:"new_sha"`
	Removed  []string `json:"removed"`
	Added    []string `json:"added"`
	NanoTime int64    `json:"time"`
}

func (d *Difference) SimpleString() string {
	return u.Format("%s %s %s", d.FullName, d.RefData, time.Unix(0, d.NanoTime).Format(time.RFC3339))
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
		if resp, err := dm.app.index.GetByID("dependency", dep); err != nil || !resp.Found {
			return dep
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
	added := []string{}
	removed := []string{}
	table.Fill("Removed")
	table.Fill("Added")
	for i := 0; i < height; i++ {
		if i < len(d.Removed) {
			removed = append(removed, getDep(d.Removed[i]))
		}
		if i < len(d.Added) {
			added = append(added, getDep(d.Added[i]))
		}
	}
	sort.Strings(removed)
	sort.Strings(added)
	for len(removed) < height {
		removed = append(removed, "")
	}
	for len(added) < height {
		added = append(added, "")
	}
	for i := 0; i < height; i++ {
		table.Fill(removed[i])
		table.Fill(added[i])
	}
	return u.Format("Repository %s %s from\n%s -> %s\n%s", d.FullName, d.RefData, d.OldSha, d.NewSha, table.Format().NoRowBorders().SpaceAllColumns().String())
}

func (d *DifferenceManager) AllDiffs() (*[]Difference, error) {
	resp, err := es.MatchAllSize(d.app.index, "difference", d.app.searchSize)
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

func (d *DifferenceManager) DiffList() ([]string, error) {
	diffs, err := d.AllDiffs()
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

	var oldDeps, newDeps []es.Dependency
	errs := make(chan error, 2)
	go func() {
		var err error
		oldDeps, err = d.app.rtrvr.DepsByShaNameGen(fullName, oldSha)
		if err != nil {
			errs <- u.Error("Could not get old sha: %s", err.Error())
			return
		}
		errs <- nil
	}()
	go func() {
		var err error
		newDeps, err = d.app.rtrvr.DepsByShaNameGen(fullName, newSha)
		if err != nil {
			errs <- u.Error("Could not get new sha: %s", err.Error())
			return
		}
		errs <- nil
	}()
	for i := 0; i < 2; i++ {
		if err := <-errs; err != nil {
			return nil, err
		}
	}
	toStrings := func(deps []es.Dependency) []string {
		res := make([]string, len(deps), len(deps))
		for i, d := range deps {
			res[i] = d.String()
		}
		return res
	}
	return d.diffCompareWrk(fullName, "Custom", toStrings(oldDeps), toStrings(newDeps), oldSha, newSha, t)
}

func (d *DifferenceManager) webhookCompare(oldEntry, newEntry es.RepositoryEntry) (*Difference, error) {
	return d.diffCompareWrk(newEntry.RepositoryFullName, newEntry.RefName, oldEntry.Dependencies, newEntry.Dependencies, oldEntry.Sha, newEntry.Sha, newEntry.Timestamp)
}

func (d *DifferenceManager) diffCompareWrk(fullName, ref string, oldDeps, newDeps []string, oldSha, newSha string, t int64) (*Difference, error) {
	added := []string{}
	removed := []string{}

	done := make(chan bool, 2)

	go func() {
		for _, newDep := range newDeps {
			if !strscont(oldDeps, newDep) {
				added = append(added, newDep)
			}
		}
		done <- true
	}()
	go func() {
		for _, oldDep := range oldDeps {
			if !strscont(newDeps, oldDep) {
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
	diff := Difference{id, fullName, ref, oldSha, newSha, removed, added, t}
	resp, err := d.app.index.PostData("difference", id, diff)
	if err != nil {
		return nil, err
	}
	if !resp.Created {
		return nil, u.Error("Diff was not created")
	}
	return &diff, nil
}

func (d *DifferenceManager) Delete(id string) {
	d.app.index.DeleteByIDWait("difference", id)
}

func strscont(sl []string, s string) bool {
	for _, ss := range sl {
		if ss == s {
			return true
		}
	}
	return false
}
