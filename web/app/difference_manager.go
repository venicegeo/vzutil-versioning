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
	"os/exec"
	"sort"
	"strings"
	"time"

	c "github.com/venicegeo/vzutil-versioning/common"
	t "github.com/venicegeo/vzutil-versioning/common/table"
	"github.com/venicegeo/vzutil-versioning/compare/pub"
	"github.com/venicegeo/vzutil-versioning/web/es"
	"github.com/venicegeo/vzutil-versioning/web/es/types"
	u "github.com/venicegeo/vzutil-versioning/web/util"
)

type DifferenceManager struct {
	app *Application

	CurrentDisplay string
}

func NewDifferenceManager(app *Application) *DifferenceManager {
	return &DifferenceManager{app, ""}
}

const DifferenceProjectField = "project_name"

type Difference struct {
	Id          string    `json:"id"`
	RepoName    string    `json:"repo_name"`
	ProjectName string    `json:"project_name"`
	Ref         string    `json:"ref"`
	OldSha      string    `json:"old_sha"`
	NewSha      string    `json:"new_sha"`
	Removed     []string  `json:"removed"`
	Added       []string  `json:"added"`
	Timestamp   time.Time `json:"time"`
}

const DifferenceMapping = `{
	"dynamic":"strict",
	"properties":{
		"id":{"type":"keyword"},
		"repo_name":{"type":"keyword"},
		"project_name":{"type":"keyword"},
		"ref":{"type":"keyword"},
		"old_sha":{"type":"keyword"},
		"new_sha":{"type":"keyword"},
		"removed":{"type":"keyword"},
		"added":{"type":"keyword"},
		"time":{"type":"keyword"}
	}
}`

func (d *Difference) SimpleString() string {
	return u.Format("%s %s %s", d.RepoName, strings.TrimPrefix(d.Ref, "refs/"), d.Timestamp.String())
}

type diffSort []Difference

func (d diffSort) Len() int {
	return len(d)
}
func (d diffSort) Swap(i, j int) {
	d[i], d[j] = d[j], d[i]
}
func (d diffSort) Less(i, j int) bool {
	return d[i].Timestamp.After(d[j].Timestamp)
}

func (dm *DifferenceManager) GenerateReport(d *Difference) string {
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
			removed = append(removed, d.Removed[i])
		}
		if i < len(d.Added) {
			added = append(added, d.Added[i])
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
	return u.Format("Repository %s %s from\n%s -> %s\n%s", d.RepoName, strings.TrimPrefix(d.Ref, "refs/"), d.OldSha, d.NewSha, table.Format().NoRowBorders().SpaceAllColumns().String())
}

func (d *DifferenceManager) GetAllDiffsInProject(proj string) (*[]Difference, error) {
	hits, err := es.GetAll(d.app.index, DifferenceType, es.NewTerm(DifferenceProjectField, proj))
	if err != nil {
		return nil, err
	}
	diffs := make([]Difference, hits.TotalHits)
	for i, hit := range hits.Hits {
		var diff Difference
		if err = json.Unmarshal(*hit.Source, &diff); err != nil {
			return nil, err
		}
		diffs[i] = diff
	}
	sort.Sort(diffSort(diffs))
	return &diffs, nil
}

func (d *DifferenceManager) ShaCompare(repoName string, files []string, oldSha, newSha string) (*Difference, error) {
	ret := make(chan *types.Scan, 2)
	defer func() {
		close(ret)
	}()
	repo := &types.Repository{Fullname: repoName, DependencyInfo: types.RepositoryDependencyInfo{repoName, types.IncomingSha, "", files}}
	d.app.wrkr.AddTask(&SingleRunnerRequest{&Repository{nil, nil, repo}, oldSha, ""}, nil, ret)
	d.app.wrkr.AddTask(&SingleRunnerRequest{&Repository{nil, nil, repo}, newSha, ""}, nil, ret)
	oldScan := <-ret
	newScan := <-ret
	if oldScan == nil || newScan == nil {
		return nil, u.Error("At least one of the shas failed")
	}
	return d.diffCompareWrk(repoName, "", "", oldScan.Scan, newScan.Scan, oldSha, newSha, time.Now(), false)
}

func (d *DifferenceManager) webhookCompare(repoName, projectName, ref string, oldEntry, newEntry *types.Scan) (*Difference, error) {
	return d.diffCompareWrk(repoName, projectName, ref, oldEntry.Scan, newEntry.Scan, oldEntry.Sha, newEntry.Sha, time.Now(), true)
}

func (d *DifferenceManager) diffCompareWrk(repoName, projectName, ref string, oldScan, newScan *c.DependencyScan, oldSha, newSha string, t time.Time, post bool) (*Difference, error) {
	oldMap := map[string]*c.DependencyScan{
		repoName: oldScan,
	}
	newMap := map[string]*c.DependencyScan{
		repoName: newScan,
	}
	oldMapDat, err := json.Marshal(oldMap)
	if err != nil {
		return nil, err
	}
	newMapDat, err := json.Marshal(newMap)
	if err != nil {
		return nil, err
	}
	dat, err := exec.Command(d.app.compareLocation, "--es", u.Format(`'%s'`, strings.TrimSpace(string(oldMapDat))), "--as", u.Format(`'%s'`, strings.TrimSpace(string(newMapDat))), "--f", "json").CombinedOutput()
	if err != nil {
		return nil, u.Error("%s, %s", err.Error(), string(dat))
	}
	comp := []*compare.CompareStruct{}
	if err = json.Unmarshal(dat, &comp); err != nil {
		return nil, err
	}
	if len(comp) != 1 {
		return nil, u.Error("Length of result was %d", len(comp))
	}
	c := comp[0]
	removed := c.ExpectedExtra
	added := c.ExpectedMissing

	if len(added) == 0 && len(removed) == 0 {
		return nil, nil
	}
	id := u.Hash(u.Format("%s%d", repoName, t))
	diff := Difference{id, repoName, projectName, ref, oldSha, newSha, removed, added, t}
	if post {
		resp, err := d.app.index.PostDataWait("difference", id, diff)
		if err != nil {
			return nil, err
		}
		if !resp.Created {
			return nil, u.Error("Diff was not created")
		}
	}
	return &diff, nil
}

func (d *DifferenceManager) Delete(id string) {
	d.app.index.DeleteByIDWait("difference", id)
}
