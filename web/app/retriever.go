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
	"strings"
	"sync"

	"github.com/venicegeo/pz-gocommon/elasticsearch"
	nt "github.com/venicegeo/pz-gocommon/gocommon"
	"github.com/venicegeo/vzutil-versioning/web/es"
	"github.com/venicegeo/vzutil-versioning/web/es/types"
	u "github.com/venicegeo/vzutil-versioning/web/util"
)

type Retriever struct {
	app *Application
}

type Project struct {
	index elasticsearch.IIndex
	*types.Project
}
type Repository struct {
	index   elasticsearch.IIndex
	project *Project
	*types.Repository
}

func NewRetriever(app *Application) *Retriever {
	return &Retriever{app}
}

//Test: TestGetScans
func (p *Project) ScanBySha(sha string) (*types.Scan, bool, error) {
	var entry = new(types.Scan)
	var err error
	//	var found bool

	result, err := p.index.GetByID(RepositoryEntry_QType, sha+"-"+p.Id)
	if result == nil {
		return nil, false, err
	} else if !result.Found {
		return nil, false, nil
	}
	return entry, true, json.Unmarshal(*result.Source, entry)
}

func (r *Retriever) ScanByShaNameGen(repo *Repository, sha string) (*types.Scan, error) {
	scan, found, err := repo.project.ScanBySha(sha)
	if err != nil || !found {
		{
			code, _, _, err := nt.HTTP(nt.HEAD, u.Format("https://github.com/%s/commit/%s", repo.Fullname, sha), nt.NewHeaderBuilder().GetHeader(), nil)
			if err != nil {
				return nil, u.Error("Could not verify this sha: %s", err.Error())
			}
			if code != 200 {
				return nil, u.Error("Could not verify this sha, head code: %d", code)
			}
		}
		exists := make(chan *types.Scan, 1)
		ret := make(chan *types.Scan, 1)
		r.app.wrkr.AddTask(&SingleRunnerRequest{
			repository: repo,
			sha:        sha,
			ref:        "",
		}, exists, ret)
		if <-exists == nil {
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

func (project *Project) ScansByRefInProject(ref string) (map[string]*types.Scan, error) {
	repos, err := project.GetAllRepositories()
	if err != nil {
		return nil, err
	}
	res := map[string]*types.Scan{}
	query := map[string]interface{}{"query": map[string]interface{}{}}
	query["query"] = map[string]interface{}{"bool": es.NewBool().
		SetMust(
			es.NewBoolQ(
				es.NewTerm(types.Scan_QField_Refs, "refs/"+ref),
				es.NewTerm(types.Scan_QField_Fullname, "%s")))}
	query["sort"] = map[string]interface{}{
		types.Scan_QField_Timestamp: map[string]interface{}{
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
		res[repoName] = &types.Scan{RepoFullname: repoName, ProjectId: project.Id, Sha: err}
		wg.Done()
		mux.Unlock()
	}
	work := func(repoName string) {
		q := u.Format(string(dat), repoName)
		var i map[string]interface{}
		json.Unmarshal([]byte(q), &i)
		resp, err := project.index.SearchByJSON(RepositoryEntry_QType, i)
		if err != nil {
			addError(repoName, u.Format("Error during query: %s", err.Error()))
			return
		}
		if resp.Hits.TotalHits == 0 {
			wg.Done()
			return
		}

		var entry = new(types.Scan)
		if err = json.Unmarshal(*resp.Hits.Hits[0].Source, entry); err != nil {
			addError(repoName, "Couldnt get entry: "+err.Error())
			return
		}
		mux.Lock()
		res[repoName] = entry
		wg.Done()
		mux.Unlock()
	}
	for _, repo := range repos {
		go work(repo.Fullname)
	}
	wg.Wait()
	return res, nil
}

// Returns map of refs to shas of a repository in a project
func (r *Repository) MapRefToShas() (map[string][]string, int64, error) {
	boool := es.NewBool().
		SetMust(es.NewBoolQ(
			es.NewTerm(types.Scan_QField_Fullname, r.Fullname),
			es.NewTerm(types.Scan_QField_ProjectId, r.ProjectId)))
	entryDat, err := es.GetAll(r.index, RepositoryEntry_QType, map[string]interface{}{"bool": boool}, map[string]interface{}{types.Scan_QField_Timestamp: "desc"})
	if err != nil {
		return nil, 0, err
	}

	res := map[string][]string{}

	for _, entryD := range entryDat.Hits {
		entry := new(types.Scan)
		if err := json.Unmarshal(*entryD.Source, entry); err != nil {
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

//Test: TestGetRepositories
func (r *Repository) GetAllRefs() ([]string, error) {
	in := es.NewAggQuery("refs", types.Scan_QField_Refs)
	boool := es.NewBool().SetMust(es.NewBoolQ(es.NewTerm(types.Scan_QField_Fullname, r.Fullname), es.NewTerm(types.Scan_QField_ProjectId, r.project.Id)))
	in["query"] = map[string]interface{}{"bool": boool}
	//	sort.Strings(res)
	resp, err := r.index.SearchByJSON(RepositoryEntry_QType, in)
	return es.GetAggKeysFromSearchResponse("refs", resp, err, func(a string) string { return strings.TrimPrefix(a, "refs/") })
}

//Test: TestGetRepositories
func (p *Project) GetAllRefs() ([]string, error) {
	repos, err := p.GetAllRepositories()
	if err != nil {
		return nil, err
	}
	reposStr := make([]string, len(repos), len(repos))
	for i, repo := range repos {
		reposStr[i] = repo.Fullname
	}
	boool := es.NewBool().SetMust(es.NewBoolQ(es.NewTerms(types.Scan_QField_Fullname, reposStr...)))
	query := es.NewAggQuery("refs", types.Scan_QField_Refs)
	query["query"] = map[string]interface{}{"bool": boool}
	resp, err := p.index.SearchByJSON(RepositoryEntry_QType, query)
	return es.GetAggKeysFromSearchResponse("refs", resp, err, func(a string) string { return strings.TrimPrefix(a, "refs/") })
}

//Test: TestGetRepositories
func (r *Retriever) ListRepositories() ([]string, error) {
	agg := es.NewAggQuery("repo", types.Scan_QField_Fullname)
	resp, err := r.app.index.SearchByJSON(RepositoryEntry_QType, agg)
	return es.GetAggKeysFromSearchResponse("repo", resp, err)
}

//Test: TestAddRepositories
func (p *Project) GetAllRepositories() ([]*Repository, error) {
	hits, err := es.GetAll(p.index, Repository_QType, es.NewTerm(types.Repository_QField_ProjectId, p.Id))
	if err != nil {
		return nil, err
	}
	res := make([]*Repository, hits.TotalHits, hits.TotalHits)
	for i, hitData := range hits.Hits {
		r := new(types.Repository)
		if err = json.Unmarshal(*hitData.Source, r); err != nil {
			return nil, err
		}
		res[i] = &Repository{p.index, p, r}
	}
	return res, nil
}

//Test: TestGetRepositories
func (p *Project) GetRepository(repository string) (*Repository, error) {
	boolq := es.NewBool().
		SetMust(es.NewBoolQ(
			es.NewTerm(types.Repository_QField_ProjectId, p.Id),
			es.NewTerm(types.Repository_QField_Name, repository)))
	resp, err := p.index.SearchByJSON(Repository_QType, map[string]interface{}{
		"query": map[string]interface{}{"bool": boolq},
	})
	if err != nil {
		return nil, err
	}
	if len(resp.Hits.Hits) != 1 {
		return nil, u.Error("Total hits not 1 but %d", len(resp.Hits.Hits))
	}
	res := new(Repository)
	if err = json.Unmarshal(*resp.Hits.Hits[0].Source, res); err != nil {
		return nil, err
	}
	res.project = p
	res.index = p.index
	return res, nil
}

//Test: TestAddRepositories
func (r *Retriever) GetRepository(repository, projectId string) (*Repository, *Project, error) {
	proj, err := r.GetProjectById(projectId)
	if err != nil {
		return nil, nil, err
	}
	repo, err := proj.GetRepository(repository)
	return repo, proj, err
}

//Test: TestAddProjects
func (r *Retriever) GetProjectById(id string) (*Project, error) {
	resp, err := r.app.index.GetByID(Project_QType, id)
	if err != nil {
		return nil, err
	} else if !resp.Found {
		return nil, u.Error("Project %s does not exist", id)
	}
	p := new(types.Project)
	if err = json.Unmarshal(*resp.Source, p); err != nil {
		return nil, err
	}
	return &Project{r.app.index, p}, nil
}

func (r *Retriever) GetRepositoryById(id string) (*Repository, error) {
	resp, err := r.app.index.GetByID(Repository_QType, id)
	if err != nil {
		return nil, err
	} else if !resp.Found {
		return nil, u.Error("Repository %s does not exist", id)
	}
	re := new(types.Repository)
	if err = json.Unmarshal(*resp.Source, re); err != nil {
		return nil, err
	}
	return &Repository{r.app.index, nil, re}, nil
}

//Test: TestAddProjects
func (r *Retriever) GetAllProjects() ([]*Project, error) {
	hits, err := es.GetAll(r.app.index, Project_QType, map[string]interface{}{})
	if err != nil {
		return nil, err
	}
	res := make([]*Project, hits.TotalHits, hits.TotalHits)
	for i, hitData := range hits.Hits {
		t := new(types.Project)
		if err = json.Unmarshal(*hitData.Source, t); err != nil {
			return nil, err
		}
		res[i] = &Project{r.app.index, t}
	}
	return res, nil
}

//Test: TestAddRepositories
func (r *Retriever) GetAllProjectNamesUsingRepository(repo string) ([]string, error) {
	agg := es.NewAggQuery("projects", types.Repository_QField_ProjectId)
	agg["query"] = es.NewTerm("repo", repo)
	resp, err := r.app.index.SearchByJSON(Repository_QType, agg)
	return es.GetAggKeysFromSearchResponse("projects", resp, err)
}
