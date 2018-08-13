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

	"github.com/venicegeo/pz-gocommon/elasticsearch"
	nt "github.com/venicegeo/pz-gocommon/gocommon"
	c "github.com/venicegeo/vzutil-versioning/common"
	"github.com/venicegeo/vzutil-versioning/web/es"
	u "github.com/venicegeo/vzutil-versioning/web/util"
)

type Retriever struct {
	app *Application
}

type Project struct {
	index *elasticsearch.Index
	*es.Project
}
type Repository struct {
	index   *elasticsearch.Index
	project *Project
	*es.ProjectEntry
}

func NewRetriever(app *Application) *Retriever {
	return &Retriever{app}
}

func (r *Retriever) ScanBySha(sha, projectRequesting string) (*c.DependencyScan, bool, error) {
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
func (r *Retriever) ScanByShaNameGen(fullName, sha, projectRequesting string) (*c.DependencyScan, error) {
	scan, found, err := r.app.rtrvr.ScanBySha(sha, projectRequesting)
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
		r.app.wrkr.AddTaskRequest(&SingleRunnerRequest{
			Fullname:  fullName,
			Sha:       sha,
			Ref:       "",
			Requester: projectRequesting,
		}, exists, ret)
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

func (r *Retriever) ScansByRefInProject(proj, ref string) (c.DependencyScans, error) {
	project, err := r.GetProject(proj)
	if err != nil {
		return nil, err
	}
	repos, err := project.GetAllRepositories()
	if err != nil {
		return nil, err
	}
	res := c.DependencyScans{}
	query := map[string]interface{}{"query": map[string]interface{}{}}
	query["query"] = map[string]interface{}{"bool": es.NewBool().
		SetMust(
			es.NewBoolQ(
				es.NewTerm(c.RefsField, "refs/"+ref),
				es.NewTerm(c.FullNameField, "%s")))}
	query["sort"] = map[string]interface{}{
		c.TimestampField: map[string]interface{}{
			"order": "desc",
		},
	}
	query["size"] = 1

	dat, err := json.MarshalIndent(query, " ", "   ")
	if err != nil {
		return nil, err
	}
	wg := sync.WaitGroup{}
	wg.Add(len(repos))
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
		if resp.Hits.TotalHits != 1 {
			wg.Done()
			return
		}

		var entry c.DependencyScan
		if err = json.Unmarshal(*resp.Hits.Hits[0].Source, &entry); err != nil {
			addError(repoName, "Couldnt get entry: "+err.Error())
			return
		}
		mux.Lock()
		res[repoName] = entry
		wg.Done()
		mux.Unlock()
	}
	for _, repo := range repos {
		go work(repo.RepoFullname)
	}
	wg.Wait()
	return res, nil
}

// Returns map of refs to shas of a repository in a project
func (r *Retriever) ListShasByRefOfRepoInProject(fullName, projectRequesting string) (map[string][]string, int64, error) {
	entryDat, err := es.GetAll(r.app.index, "repository_entry", u.Format(`{
	"bool": {
		"must": [
			{
				"term":{
					"%s":"%s"
				}
			},{
				"term":{
					"%s":"%s"
				}
			}
		]
	}
}`, c.FullNameField, fullName, c.RequesterField, projectRequesting), u.Format(`{"%s":"desc"}`, c.TimestampField))
	if err != nil {
		return nil, 0, err
	}

	res := map[string][]string{}

	for _, entryD := range entryDat.Hits {
		var entry c.DependencyScan
		if err := json.Unmarshal(*entryD.Source, &entry); err != nil {
			return nil, 0, err
		}
		for _, refName := range entry.Refs {
			if _, ok := res[refName]; !ok {
				res[refName] = []string{}
			}
			res[refName] = append(res[refName], entry.Sha)
		}
	}
	return res, entryDat.TotalHits, nil
}

//// Lists the refs of a repo within a project
//func (r *Retriever) ListRefsOfRepoInProject(fullName, projectRequesting string) ([]string, error) {
//	in := es.NewAggQuery("refs", c.RefsField)
//	boool := es.NewBool().SetMust(es.NewBoolQ(es.NewTerm(c.FullNameField, fullName), es.NewTerm(c.RequesterField, projectRequesting)))
//	in["query"] = map[string]interface{}{"bool": boool}
//	var out es.AggResponse
//	if err := r.app.index.DirectAccess("GET", "/versioning_tool/repository_entry/_search", in, &out); err != nil {
//		return nil, err
//	}

//	res := make([]string, len(out.Aggs["refs"].Buckets), len(out.Aggs["refs"].Buckets))
//	for i, r := range out.Aggs["refs"].Buckets {
//		res[i] = strings.TrimPrefix(r.Key, `refs/`)
//	}
//	sort.Strings(res)
//	return res, nil
//}

func (r *Repository) GetAllRefs() ([]string, error) {
	in := es.NewAggQuery("refs", c.RefsField)
	boool := es.NewBool().SetMust(es.NewBoolQ(es.NewTerm(c.FullNameField, r.RepoFullname), es.NewTerm(c.RequesterField, r.project.Name)))
	in["query"] = map[string]interface{}{"bool": boool}
	var out es.AggResponse
	if err := r.index.DirectAccess("GET", "/versioning_tool/repository_entry/_search", in, &out); err != nil {
		return nil, err
	}

	res := make([]string, len(out.Aggs["refs"].Buckets), len(out.Aggs["refs"].Buckets))
	for i, r := range out.Aggs["refs"].Buckets {
		res[i] = strings.TrimPrefix(r.Key, `refs/`)
	}
	sort.Strings(res)
	return res, nil
}

func (p *Project) GetAllRefs() ([]string, error) {
	repos, err := p.GetAllRepositories()
	if err != nil {
		return nil, err
	}
	reposStr := make([]string, len(repos), len(repos))
	for i, repo := range repos {
		reposStr[i] = repo.RepoFullname
	}
	boool := es.NewBool().SetMust(es.NewBoolQ(es.NewTerms(c.FullNameField, reposStr...)))
	query := es.NewAggQuery("refs", c.RefsField)
	query["query"] = map[string]interface{}{"bool": boool}
	var out es.AggResponse
	if err := p.index.DirectAccess("GET", "/versioning_tool/repository_entry/_search", query, &out); err != nil {
		return nil, err
	}
	res := make([]string, len(out.Aggs["refs"].Buckets), len(out.Aggs["refs"].Buckets))
	for i, bucket := range out.Aggs["refs"].Buckets {
		res[i] = strings.TrimPrefix(bucket.Key, "refs/")
	}
	sort.Strings(res)
	return res, nil
}

// Returns map of repository to all of its refs, within a project
func (r *Retriever) ListRefsByRepositoryInProject(projectRequesting string) (*map[string][]string, int, error) {
	project, err := r.GetProject(projectRequesting)
	if err != nil {
		return nil, 0, err
	}
	repos, err := project.GetAllRepositories()
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
		go work(p.RepoFullname)
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

// Lists all known repositories, regardless of project
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

//// Lists all repositories within a project
//func (r *Retriever) ListRepositoriesInProject(projectRequesting string) ([]string, error) {
//	exists, err := r.app.index.ItemExists("project", projectRequesting)
//	if err != nil {
//		return nil, err
//	} else if !exists {
//		return nil, u.Error("Project %s does not exist", err.Error())
//	}
//	hits, err := es.GetAll(r.app.index, "project_entry", u.Format(`{
//	"term": {
//		"%s":"%s"
//	}
//}`, "project_name", projectRequesting))
//	if err != nil {
//		return nil, err
//	}
//	res := make([]string, hits.TotalHits, hits.TotalHits)
//	for i, hitData := range hits.Hits {
//		t := new(es.ProjectEntry)
//		if err = json.Unmarshal(*hitData.Source, t); err != nil {
//			return nil, err
//		}
//		res[i] = t.RepoFullname
//	}
//	return res, nil
//}

func (p *Project) GetAllRepositories() ([]*Repository, error) {
	hits, err := es.GetAll(p.index, "project_entry", u.Format(`{
	"term": {
		"%s":"%s"
	}
}`, "project_name", p.Name))
	if err != nil {
		return nil, err
	}
	res := make([]*Repository, hits.TotalHits, hits.TotalHits)
	for i, hitData := range hits.Hits {
		r := new(es.ProjectEntry)
		if err = json.Unmarshal(*hitData.Source, r); err != nil {
			return nil, err
		}
		res[i] = &Repository{p.index, p, r}
	}
	return res, nil
}

func (r *Retriever) GetProject(name string) (*Project, error) {
	resp, err := r.app.index.GetByID("project", name)
	if err != nil {
		return nil, err
	} else if !resp.Found {
		return nil, u.Error("Project %s does not exist", err.Error())
	}
	p := new(es.Project)
	if err = json.Unmarshal(*resp.Source, p); err != nil {
		return nil, err
	}
	return &Project{r.app.index, p}, nil
}

func (r *Retriever) GetAllProjects() ([]*Project, error) {
	hits, err := es.GetAll(r.app.index, "project", "{}")
	if err != nil {
		return nil, err
	}
	res := make([]*Project, hits.TotalHits, hits.TotalHits)
	for i, hitData := range hits.Hits {
		t := new(es.Project)
		if err = json.Unmarshal(*hitData.Source, t); err != nil {
			return nil, err
		}
		res[i] = &Project{r.app.index, t}
	}
	return res, nil

}

// Lists all known projects
//func (r *Retriever) ListProjects() ([]*es.Project, error) {
//	hits, err := es.GetAll(r.app.index, "project", "{}")
//	if err != nil {
//		return nil, err
//	}
//	res := make([]*es.Project, hits.TotalHits, hits.TotalHits)
//	for i, hitData := range hits.Hits {
//		t := new(es.Project)
//		if err = json.Unmarshal(*hitData.Source, t); err != nil {
//			return nil, err
//		}
//		res[i] = t
//	}
//	return res, nil
//}
