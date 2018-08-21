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
	u "github.com/venicegeo/vzutil-versioning/web/util"
)

type Retriever struct {
	app *Application
}

type Project struct {
	index elasticsearch.IIndex
	*es.Project
}
type Repository struct {
	index   elasticsearch.IIndex
	project *Project
	*es.ProjectEntry
}

func NewRetriever(app *Application) *Retriever {
	return &Retriever{app}
}

//Test: TestGetScans
func (p *Project) ScanBySha(sha string) (*RepositoryDependencyScan, bool, error) {
	var entry = new(RepositoryDependencyScan)
	var err error
	//	var found bool

	result, err := p.index.GetByID(RepositoryEntryType, sha+"-"+p.Name)
	if result == nil {
		return nil, false, err
	} else if !result.Found {
		return nil, false, nil
	}
	return entry, true, json.Unmarshal(*result.Source, entry)
}

func (r *Retriever) ScanByShaNameGen(repo *Repository, sha string) (*RepositoryDependencyScan, error) {
	scan, found, err := repo.project.ScanBySha(sha)
	if err != nil || !found {
		{
			code, _, _, err := nt.HTTP(nt.HEAD, u.Format("https://github.com/%s/commit/%s", repo.RepoFullname, sha), nt.NewHeaderBuilder().GetHeader(), nil)
			if err != nil {
				return nil, u.Error("Could not verify this sha: %s", err.Error())
			}
			if code != 200 {
				return nil, u.Error("Could not verify this sha, head code: %d", code)
			}
		}
		exists := make(chan bool, 1)
		ret := make(chan *RepositoryDependencyScan, 1)
		r.app.wrkr.AddTask(&SingleRunnerRequest{
			repository: repo,
			sha:        sha,
			ref:        "",
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

func (project *Project) ScansByRefInProject(ref string) (map[string]*RepositoryDependencyScan, error) {
	repos, err := project.GetAllRepositories()
	if err != nil {
		return nil, err
	}
	res := map[string]*RepositoryDependencyScan{}
	query := map[string]interface{}{"query": map[string]interface{}{}}
	query["query"] = map[string]interface{}{"bool": es.NewBool().
		SetMust(
			es.NewBoolQ(
				es.NewTerm(Scan_RefsField, "refs/"+ref),
				es.NewTerm(Scan_FullnameField, "%s")))}
	query["sort"] = map[string]interface{}{
		Scan_TimestampField: map[string]interface{}{
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
		res[repoName] = &RepositoryDependencyScan{RepoFullname: repoName, Project: project.DisplayName, Sha: err}
		wg.Done()
		mux.Unlock()
	}
	work := func(repoName string) {
		q := u.Format(string(dat), repoName)
		var i map[string]interface{}
		json.Unmarshal([]byte(q), &i)
		resp, err := project.index.SearchByJSON(RepositoryEntryType, i)
		if err != nil {
			addError(repoName, u.Format("Error during query: %s", err.Error()))
			return
		}
		if resp.Hits.TotalHits != 1 {
			wg.Done()
			return
		}

		var entry = new(RepositoryDependencyScan)
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
		go work(repo.RepoFullname)
	}
	wg.Wait()
	return res, nil
}

// Returns map of refs to shas of a repository in a project
func (r *Repository) MapRefToShas() (map[string][]string, int64, error) {
	boool := es.NewBool().
		SetMust(es.NewBoolQ(
			es.NewTerm(Scan_FullnameField, r.RepoFullname),
			es.NewTerm(Scan_ProjectField, r.ProjectName)))
	entryDat, err := es.GetAll(r.index, RepositoryEntryType, map[string]interface{}{"bool": boool}, map[string]interface{}{Scan_TimestampField: "desc"})
	if err != nil {
		return nil, 0, err
	}

	res := map[string][]string{}

	for _, entryD := range entryDat.Hits {
		entry := new(RepositoryDependencyScan)
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
	in := es.NewAggQuery("refs", Scan_RefsField)
	boool := es.NewBool().SetMust(es.NewBoolQ(es.NewTerm(Scan_FullnameField, r.RepoFullname), es.NewTerm(Scan_ProjectField, r.project.Name)))
	in["query"] = map[string]interface{}{"bool": boool}
	//	sort.Strings(res)
	resp, err := r.index.SearchByJSON(RepositoryEntryType, in)
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
		reposStr[i] = repo.RepoFullname
	}
	boool := es.NewBool().SetMust(es.NewBoolQ(es.NewTerms(Scan_FullnameField, reposStr...)))
	query := es.NewAggQuery("refs", Scan_RefsField)
	query["query"] = map[string]interface{}{"bool": boool}
	resp, err := p.index.SearchByJSON(RepositoryEntryType, query)
	return es.GetAggKeysFromSearchResponse("refs", resp, err, func(a string) string { return strings.TrimPrefix(a, "refs/") })
}

//Test: TestGetRepositories
func (r *Retriever) ListRepositories() ([]string, error) {
	agg := es.NewAggQuery("repo", Scan_FullnameField)
	resp, err := r.app.index.SearchByJSON(RepositoryEntryType, agg)
	return es.GetAggKeysFromSearchResponse("repo", resp, err)
}

//Test: TestAddRepositories
func (p *Project) GetAllRepositories() ([]*Repository, error) {
	hits, err := es.GetAll(p.index, ProjectEntryType, es.NewTerm(es.ProjectEntryNameField, p.Name))
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

//Test: TestGetRepositories
func (p *Project) GetRepository(repository string) (*Repository, error) {
	boolq := es.NewBool().
		SetMust(es.NewBoolQ(
			es.NewTerm(es.ProjectEntryNameField, p.Name),
			es.NewTerm(es.ProjectEntryRepositoryField, repository)))
	resp, err := p.index.SearchByJSON(ProjectEntryType, map[string]interface{}{
		"query": map[string]interface{}{"bool": boolq},
	})
	if err != nil {
		return nil, err
	}
	if resp.TotalHits() != 1 {
		return nil, u.Error("Total hits not 1 but %d", resp.TotalHits())
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
func (r *Retriever) GetRepository(repository, project string) (*Repository, *Project, error) {
	proj, err := r.GetProject(project)
	if err != nil {
		return nil, nil, err
	}
	repo, err := proj.GetRepository(repository)
	return repo, proj, err
}

//Test: TestAddProjects
func (r *Retriever) GetProject(name string) (*Project, error) {
	resp, err := r.app.index.GetByID(ProjectType, name)
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

//Test: TestAddProjects
func (r *Retriever) GetAllProjects() ([]*Project, error) {
	hits, err := es.GetAll(r.app.index, ProjectType, map[string]interface{}{})
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

//Test: TestAddRepositories
func (r *Retriever) GetAllProjectNamesUsingRepository(repo string) ([]string, error) {
	agg := es.NewAggQuery("projects", es.ProjectEntryNameField)
	agg["query"] = es.NewTerm("repo", repo)
	resp, err := r.app.index.SearchByJSON(ProjectEntryType, agg)
	return es.GetAggKeysFromSearchResponse("projects", resp, err)
}
