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
	"fmt"
	"sort"
	"strings"
	"sync"

	nt "github.com/venicegeo/pz-gocommon/gocommon"
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

func (r *Retriever) DepsBySha(sha string) ([]es.Dependency, string, string, bool, error) {
	var entry es.RepositoryEntry
	var err error
	//	var found bool

	result, err := r.app.index.GetByID("repository_entry", sha)
	if result == nil {
		return nil, "", "", false, err
	} else if !result.Found {
		return nil, "", "", false, nil
	}
	if err = json.Unmarshal(*result.Source, &entry); err != nil {
		return nil, "", "", true, err
	}

	return r.depsFromEntry(&entry), entry.RepositoryFullName, entry.RefName, true, nil
}
func (r *Retriever) DepsByShaNameGen(fullName, sha string) ([]es.Dependency, error) {
	deps, _, _, found, err := r.app.rtrvr.DepsBySha(sha)
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
		ret := make(chan *SingleResult, 1)
		r.app.wrkr.AddTask(&s.GitWebhook{AfterSha: sha, Repository: s.GitRepository{FullName: fullName}}, exists, ret)
		if !<-exists {
			sr := <-ret
			if sr == nil {
				return nil, u.Error("There was an error while running this")
			}
			deps = sr.Deps
			sort.Sort(es.DependencySort(deps))
		} else {
			return nil, u.Error("Retriever said not found, worker said found")
		}
	}
	return deps, nil
}

func (r *Retriever) depsFromEntry(entry *es.RepositoryEntry) (res []es.Dependency) {
	mux := &sync.Mutex{}
	done := make(chan bool, len(entry.Dependencies))
	work := func(dep string) {
		if resp, err := r.app.index.GetByID("dependency", dep); err != nil || !resp.Found {
			name := u.Format("Cound not find [%s]", dep)
			tmp := es.Dependency{name, "", ""}
			mux.Lock()
			res = append(res, tmp)
			mux.Unlock()
		} else {
			var depen es.Dependency
			if err = json.Unmarshal([]byte(*resp.Source), &depen); err != nil {
				tmp := es.Dependency{u.Format("Error getting [%s]: [%s]", dep, err.Error()), "", ""}
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
	for _, d := range entry.Dependencies {
		go work(d)
	}
	for i := 0; i < len(entry.Dependencies); i++ {
		<-done
	}
	sort.Sort(es.DependencySort(res))
	return res
}

func (r *Retriever) DepsByRefInProject(proj, ref string) (ReportByRefS, error) {
	repoNames, err := r.ListRepositoriesByProj(proj)
	if err != nil {
		return nil, err
	}
	return r.byRefWork(repoNames, ref)
}

type ReportByRefS map[string]reportByRefS
type reportByRefS struct {
	deps []es.Dependency
	sha  string
}

func (r *Retriever) byRefWork(repoNames []string, ref string) (res ReportByRefS, err error) {
	res = ReportByRefS{}
	query := map[string]interface{}{}
	query["query"] = map[string]interface{}{
		"bool": map[string]interface{}{
			"must": []map[string]interface{}{
				map[string]interface{}{
					"term": map[string]interface{}{
						"ref_name": "refs/" + ref,
					},
				},
				map[string]interface{}{
					"term": map[string]interface{}{
						"repo_fullname": "%s",
					},
				},
			},
		},
	}
	query["sort"] = map[string]interface{}{
		"timestamp": map[string]interface{}{
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
		res[repoName] = reportByRefS{[]es.Dependency{es.Dependency{Version: err}}, "Unknown"}
		wg.Done()
		mux.Unlock()
	}
	work := func(repoName string) {
		q := []byte(fmt.Sprintf(string(dat), repoName))
		resp, err := r.app.index.SearchByJSON("repository_entry", string(q))
		if err != nil {
			addError(repoName, u.Format("Error during query: %s", err.Error()))
			return
		}
		if resp.NumHits() != 1 {
			wg.Done()
			return
		}

		var entry es.RepositoryEntry
		if err = json.Unmarshal(*resp.GetHit(0).Source, &entry); err != nil {
			addError(repoName, "Couldnt get entry: "+err.Error())
			return
		}
		mux.Lock()
		res[repoName] = reportByRefS{r.depsFromEntry(&entry), entry.Sha}
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
		"repo_fullname":"%s"
	}
}`, fullName), `{"timestamp":"desc"}`)
	if err != nil {
		return nil, 0, err
	}

	res := map[string][]string{}

	for _, entryD := range entryDat {
		var entry es.RepositoryEntry
		if err := json.Unmarshal(entryD.Dat, &entry); err != nil {
			return nil, 0, err
		}
		if _, ok := res[entry.RefName]; !ok {
			res[entry.RefName] = []string{}
		}
		res[entry.RefName] = append(res[entry.RefName], entry.Sha)
	}
	return res, len(entryDat), nil
}

func (r *Retriever) ListRefsRepo(fullName string) ([]string, error) {
	in := es.NewAggQuery("refs", "ref_name")
	in["size"] = 0
	in["query"] = map[string]interface{}{
		"term": map[string]interface{}{
			"repo_fullname": fullName,
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
	boolQ := es.NewBoolQ()
	for _, repo := range repos {
		boolQ.Add(es.NewTerm("repo_fullname", repo))
	}
	boool := es.NewBool().SetShould(boolQ)
	query := es.NewAggQuery("refs", "ref_name")
	query["query"] = map[string]interface{}{"bool": boool}
	var out es.AggResponse
	if err := r.app.index.DirectAccess("GET", "/versioning_tool/repository_entry/_search", query, &out); err != nil {
		return nil, err
	}
	res := make([]string, len(out.Aggs["refs"].Buckets), len(out.Aggs["refs"].Buckets))
	for i, bucket := range out.Aggs["refs"].Buckets {
		res[i] = strings.TrimPrefix(bucket.Key, "refs/")
	}
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
		in := es.NewAggQuery("refs", "ref_name")
		in["query"] = map[string]interface{}{
			"term": map[string]interface{}{
				"repo_fullname": repo,
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

func (r *Retriever) ListRepositoriesByProj(proj string) ([]string, error) {
	exists, err := r.app.index.ItemExists("project", proj)
	if err != nil {
		return nil, err
	} else if !exists {
		return nil, u.Error("Project %s does not exist", err.Error())
	}
	hits, err := es.GetAll(r.app.index, "project_entry", u.Format(`{
	"term": {
		"name":"%s"
	}
}`, proj))
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
