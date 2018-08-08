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
	"strings"
	"sync"

	nt "github.com/venicegeo/pz-gocommon/gocommon"
	c "github.com/venicegeo/vzutil-versioning/common"
	s "github.com/venicegeo/vzutil-versioning/web/app/structs"
	"github.com/venicegeo/vzutil-versioning/web/es"
	u "github.com/venicegeo/vzutil-versioning/web/util"
)

type Retriever struct {
	app *Application
}

func NewRetriever(app *Application) *Retriever {
	return &Retriever{app}
}

func (r *Retriever) ScanBySha(sha string) (*c.DependencyScan, bool, error) {
	var entry c.DependencyScan
	var err error
	//	var found bool

	result, err := r.app.index.GetByID("repository_entry", sha)
	if result == nil {
		return nil, false, err
	} else if !result.Found {
		return nil, false, nil
	}
	return &entry, true, json.Unmarshal(*result.Source, &entry)
}
func (r *Retriever) ScanByShaNameGen(fullName, sha string) (*c.DependencyScan, error) {
	scan, found, err := r.app.rtrvr.ScanBySha(sha)
	if err != nil || !found {
		{
			code, _, _, err := nt.HTTP(nt.HEAD, u.Format("https://github.com/%s/commit/%s", fullName, sha), nt.NewHeaderBuilder().GetHeader(), nil)
			if err != nil {
				return nil, u.Error("Could not verify this sha: %s", err.Error())
			}
			if code != 200 {
				return nil, u.Error("Could not verify this sha, head code: %d", code)
			}
		}
		exists := make(chan bool, 1)
		ret := make(chan *c.DependencyScan, 1)
		r.app.wrkr.AddTask(&s.GitWebhook{AfterSha: sha, Repository: s.GitRepository{FullName: fullName}}, exists, ret)
		if !<-exists {
			scan = <-ret
			if scan == nil {
				return nil, u.Error("There was an error while running this")
			}
		} else {
			return nil, u.Error("Retriever said not found, worker said found")
		}
	}
	return scan, nil
}

//TODO delete?
/*
func (r *Retriever) depsFromEntry(entry *c.DependencyScan) (res []d.Dependency) {
	mux := &sync.Mutex{}
	done := make(chan bool, len(entry.Deps))
	work := func(dep string) {
		if resp, err := r.app.index.GetByID("dependency", dep); err != nil || !resp.Found {
			name := u.Format("Cound not find [%s]", dep)
			tmp := d.Dependency{name, "", ""}
			mux.Lock()
			res = append(res, tmp)
			mux.Unlock()
		} else {
			var depen d.Dependency
			if err = json.Unmarshal([]byte(*resp.Source), &depen); err != nil {
				tmp := d.Dependency{u.Format("Error getting [%s]: [%s]", dep, err.Error()), "", ""}
				mux.Lock()
				res = append(res, tmp)
				mux.Unlock()
			} else {
				mux.Lock()
				res = append(res, depen)
				mux.Unlock()
			}
		}
		done <- true
	}
	for _, d := range entry.Deps {
		go work(d)
	}
	for i := 0; i < len(entry.Deps); i++ {
		<-done
	}
	sort.Sort(d.DependencySort(res))
	return res
}
*/

func (r *Retriever) ScansByRefInProject(proj, ref string) (c.DependencyScans, error) {
	repoNames, err := r.ListRepositoriesByProj(proj)
	if err != nil {
		return nil, err
	}
	return r.byRefWork(repoNames, ref)
}

func (r *Retriever) byRefWork(repoNames []string, ref string) (res c.DependencyScans, err error) {
	res = c.DependencyScans{}
	query := map[string]interface{}{}
	query["query"] = map[string]interface{}{
		"bool": map[string]interface{}{
			"must": []map[string]interface{}{
				map[string]interface{}{
					"term": map[string]interface{}{
						c.RefsField: "refs/" + ref,
					},
				},
				map[string]interface{}{
					"term": map[string]interface{}{
						c.FullNameField: "%s",
					},
				},
			},
		},
	}
	query["sort"] = map[string]interface{}{
		c.TimestampField: map[string]interface{}{
			"order": "desc",
		},
	}
	query["size"] = 1
	//	type Hit struct {
	//		Id     string          `json:"_id"`
	//		Source json.RawMessage `json:"_source"`
	//	}
	//	type Result struct {
	//		Total int64 `json:"total"`
	//		Hits  []Hit `json:"hits"`
	//	}
	//	type Wrapper struct {
	//		R Result `json:"hits"`
	//	}
	dat, err := json.MarshalIndent(query, " ", "   ")
	if err != nil {
		return nil, err
	}
	wg := sync.WaitGroup{}
	wg.Add(len(repoNames))
	mux := sync.Mutex{}
	addError := func(repoName, err string) {
		mux.Lock()
		res[repoName] = c.DependencyScan{Fullname: repoName, Name: repoName, Sha: err}
		wg.Done()
		mux.Unlock()
	}
	work := func(repoName string) {
		q := []byte(u.Format(string(dat), repoName))
		resp, err := r.app.index.SearchByJSON("repository_entry", string(q))
		if err != nil {
			addError(repoName, u.Format("Error during query: %s", err.Error()))
			return
		}
		if resp.NumHits() != 1 {
			wg.Done()
			return
		}

		var entry c.DependencyScan
		if err = json.Unmarshal(*resp.GetHit(0).Source, &entry); err != nil {
			addError(repoName, "Couldnt get entry: "+err.Error())
			return
		}
		mux.Lock()
		res[repoName] = entry
		wg.Done()
		mux.Unlock()
	}
	for _, repoName := range repoNames {
		go work(repoName)
	}
	wg.Wait()
	return
}

func (r *Retriever) ListShas(fullName string) (map[string][]string, int, error) {
	entryDat, err := es.GetAll(r.app.index, "repository_entry", u.Format(`{
	"term":{
		"%s":"%s"
	}
}`, c.FullNameField, fullName), u.Format(`{"%s":"desc"}`, c.TimestampField))
	if err != nil {
		return nil, 0, err
	}

	res := map[string][]string{}

	for _, entryD := range entryDat {
		var entry c.DependencyScan
		if err := json.Unmarshal(entryD.Dat, &entry); err != nil {
			return nil, 0, err
		}
		for _, refName := range entry.Refs {
			if _, ok := res[refName]; !ok {
				res[refName] = []string{}
			}
			res[refName] = append(res[refName], entry.Sha)
		}
	}
	return res, len(entryDat), nil
}

func (r *Retriever) ListRefsRepo(fullName string) ([]string, error) {
	in := es.NewAggQuery("refs", c.RefsField)
	in["query"] = map[string]interface{}{
		"term": map[string]interface{}{
			c.FullNameField: fullName,
		},
	}
	var out es.AggResponse
	if err := r.app.index.DirectAccess("GET", "/versioning_tool/repository_entry/_search", in, &out); err != nil {
		return nil, err
	}

	res := make([]string, len(out.Aggs["refs"].Buckets), len(out.Aggs["refs"].Buckets))
	for i, r := range out.Aggs["refs"].Buckets {
		res[i] = strings.TrimPrefix(r.Key, `refs/`)
	}
	sort.Strings(res)
	return res, nil
}

func (r *Retriever) ListRefsInProj(proj string) ([]string, error) {
	repos, err := r.ListRepositoriesByProj(proj)
	if err != nil {
		return nil, err
	}
	boool := es.NewBool().SetMust(es.NewBoolQ(es.NewTerms(c.FullNameField, repos...)))
	query := es.NewAggQuery("refs", c.RefsField)
	query["query"] = map[string]interface{}{"bool": boool}
	var out es.AggResponse
	if err := r.app.index.DirectAccess("GET", "/versioning_tool/repository_entry/_search", query, &out); err != nil {
		return nil, err
	}
	res := make([]string, len(out.Aggs["refs"].Buckets), len(out.Aggs["refs"].Buckets))
	for i, bucket := range out.Aggs["refs"].Buckets {
		res[i] = strings.TrimPrefix(bucket.Key, "refs/")
	}
	sort.Strings(res)
	return res, nil
}

func (r *Retriever) ListRefsInProjByRepo(proj string) (*map[string][]string, int, error) {
	repos, err := r.ListRepositoriesByProj(proj)
	if err != nil {
		return nil, 0, err
	}
	res := map[string][]string{}
	totalNumber := 0
	errs := make(chan error, len(repos))
	mux := &sync.Mutex{}

	work := func(repo string) {
		in := es.NewAggQuery("refs", c.RefsField)
		in["query"] = map[string]interface{}{
			"term": map[string]interface{}{
				c.FullNameField: repo,
			},
		}
		var out es.AggResponse
		if err := r.app.index.DirectAccess("GET", "/versioning_tool/repository_entry/_search", in, &out); err != nil {
			errs <- err
			return
		}
		num := len(out.Aggs["refs"].Buckets)
		temp := make([]string, num, num)
		for i, r := range out.Aggs["refs"].Buckets {
			temp[i] = strings.TrimPrefix(r.Key, `refs/`)
		}
		sort.Strings(temp)
		mux.Lock()
		{
			totalNumber += num
			res[repo] = temp
			errs <- nil
		}
		mux.Unlock()
	}

	for _, p := range repos {
		go work(p)
	}
	err = nil
	for i := 0; i < len(repos); i++ {
		e := <-errs
		if e != nil {
			err = e
		}
	}
	return &res, totalNumber, err
}

func (r *Retriever) ListRepositories() ([]string, error) {
	agg := es.NewAggQuery("repo", c.FullNameField)
	var resp es.AggResponse
	if err := r.app.index.DirectAccess("POST", "/versioning_tool/repository_entry/_search", agg, &resp); err != nil {
		return nil, err
	}
	hits := resp.Aggs["repo"].Buckets
	res := make([]string, len(hits), len(hits))
	for i, hitData := range hits {
		res[i] = hitData.Key
	}
	return res, nil
}

func (r *Retriever) ListRepositoriesByProj(proj string) ([]string, error) {
	exists, err := r.app.index.ItemExists("project", proj)
	if err != nil {
		return nil, err
	} else if !exists {
		return nil, u.Error("Project %s does not exist", err.Error())
	}
	hits, err := es.GetAll(r.app.index, "project_entry", u.Format(`{
	"term": {
		"%s":"%s"
	}
}`, c.NameField, proj))
	if err != nil {
		return nil, err
	}
	res := make([]string, len(hits), len(hits))
	for i, hitData := range hits {
		t := new(es.ProjectEntry)
		if err = json.Unmarshal(hitData.Dat, t); err != nil {
			return nil, err
		}
		res[i] = t.Repo
	}
	return res, nil
}

func (r *Retriever) ListProjects() ([]*es.Project, error) {
	hits, err := es.GetAll(r.app.index, "project", "{}")
	if err != nil {
		return nil, err
	}
	res := make([]*es.Project, len(hits), len(hits))
	for i, hitData := range hits {
		t := new(es.Project)
		if err = json.Unmarshal(hitData.Dat, t); err != nil {
			return nil, err
		}
		res[i] = t
	}
	return res, nil
}
