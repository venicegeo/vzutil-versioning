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
	"bytes"
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

func (r *Retriever) DepsBySha(sha string) ([]es.Dependency, bool, error) {
	var entry es.RepositoryEntry
	var err error
	//	var found bool

	result, err := r.app.index.GetByID("repository_entry", sha)
	if result == nil {
		return nil, false, err
	} else if !result.Found {
		return nil, false, nil
	}
	if err = json.Unmarshal(*result.Source, &entry); err != nil {
		return nil, true, err
	}

	return r.DepsFromEntry(&entry), true, nil
}
func (r *Retriever) DepsByShaNameGen(fullName, sha string) ([]es.Dependency, error) {
	deps, found, err := r.app.rtrvr.DepsBySha(sha)
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
func (r *Retriever) DepsFromEntry(entry *es.RepositoryEntry) (res []es.Dependency) {
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

func (r *Retriever) DepsByRef(info ...string) (map[string][]es.Dependency, error) {
	switch len(info) {
	case 1: //just a ref
		repoNames, err := r.ListRepositories()
		if err != nil {
			return nil, err
		}
		return r.byRefWork(repoNames, info[0])
	case 2: //org and ref
		repoNames, err := r.ListRepositoriesByOrg(info[0])
		if err != nil {
			return nil, err
		}
		return r.byRefWork(repoNames, info[1])
	case 3: //org repo and ref
		repoNames := []string{fmt.Sprintf("%s/%s", info[0], info[1])}
		return r.byRefWork(repoNames, info[2])
	}
	return nil, u.Error("Sorry, something is wrong with the code..")
}

func (r *Retriever) byRefWork(repoNames []string, ref string) (res map[string][]es.Dependency, err error) {
	res = map[string][]es.Dependency{}
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
	type Hit struct {
		Id     string          `json:"_id"`
		Source json.RawMessage `json:"_source"`
	}
	type Result struct {
		Total int64 `json:"total"`
		Hits  []Hit `json:"hits"`
	}
	type Wrapper struct {
		R Result `json:"hits"`
	}
	dat, err := json.MarshalIndent(query, " ", "   ")
	if err != nil {
		return nil, err
	}
	wg := sync.WaitGroup{}
	wg.Add(len(repoNames))
	mux := sync.Mutex{}
	addError := func(repoName, err string) {
		mux.Lock()
		res[repoName] = []es.Dependency{es.Dependency{Version: err}}
		wg.Done()
		mux.Unlock()
	}
	work := func(repoName string) {
		q := []byte(fmt.Sprintf(string(dat), repoName))
		var out Wrapper
		code, dat, _, err := nt.HTTP(nt.GET, "http://localhost:9200/versioning_tool/repository_entry/_search", nt.NewHeaderBuilder().GetHeader(), bytes.NewReader(q))
		if err != nil {
			addError(repoName, "Error during query: "+err.Error())
			return
		} else if code != 200 {
			addError(repoName, fmt.Sprintf("Query code not 200: %d", code))
			return
		}
		d := json.NewDecoder(bytes.NewReader(dat))
		d.UseNumber()
		err = d.Decode(&out)
		if err != nil {
			addError(repoName, "Error decoding: "+err.Error())
			return
		}
		if len(out.R.Hits) != 1 {
			wg.Done()
			return
		}
		var entry es.RepositoryEntry
		if err = json.Unmarshal(out.R.Hits[0].Source, &entry); err != nil {
			addError(repoName, "Couldnt get entry: "+err.Error())
			return
		}
		mux.Lock()
		res[repoName] = r.DepsFromEntry(&entry)
		wg.Done()
		mux.Unlock()
	}
	for _, repoName := range repoNames {
		go work(repoName)
	}
	wg.Wait()
	return

}

//

func (r *Retriever) ListShas(fullName string) (map[string][]string, int, error) {
	entryDat, err := es.GetAll(r.app.index, "repository_entry", u.Format(`{
	"term":{
		"repo_fullname":"%s"
	}
}`, fullName))
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

////

func (r *Retriever) ListRefsRepo(fullName string) ([]string, error) {
	in := r.newAggQuery("refs", "ref_name")
	in["size"] = 0
	in["query"] = map[string]interface{}{
		"term": map[string]interface{}{
			"repo_fullname": fullName,
		},
	}
	var out AggResponse
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

func (r *Retriever) ListRefs(org string) (*map[string][]string, int, error) {
	repos, err := r.ListRepositoriesByOrg(org)
	if err != nil {
		return nil, 0, err
	}
	res := map[string][]string{}
	totalNumber := 0
	errs := make(chan error, len(repos))
	mux := &sync.Mutex{}

	work := func(repo string) {
		in := r.newAggQuery("refs", "ref_name")
		in["query"] = map[string]interface{}{
			"term": map[string]interface{}{
				"repo_fullname": repo,
			},
		}
		var out AggResponse
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

////

func (r *Retriever) newAggQuery(aggName, fieldName string) map[string]interface{} {
	return map[string]interface{}{
		"aggs": map[string]interface{}{
			aggName: map[string]interface{}{
				"terms": map[string]interface{}{
					"field": fieldName,
					"size":  10000,
				},
			},
		},
		"size": 0}
}

func (r *Retriever) ListRepositories() ([]string, error) {
	var out AggResponse
	in := r.newAggQuery("repos", "repo_fullname")
	err := r.app.index.DirectAccess("POST", "/versioning_tool/repository_entry/_search", in, &out)
	if err != nil {
		return nil, err
	}
	res := make([]string, len(out.Aggs["repos"].Buckets), len(out.Aggs["repos"].Buckets))
	for i, repo := range out.Aggs["repos"].Buckets {
		res[i] = repo.Key
	}
	sort.Strings(res)
	return res, nil
}

func (r *Retriever) ListRepositoriesByOrg(org string) ([]string, error) {
	all, err := r.ListRepositories()
	if err != nil {
		return nil, err
	}
	org += "/"
	res := []string{}
	for _, repo := range all {
		if strings.HasPrefix(repo, org) {
			res = append(res, repo)
		}
	}
	return res, nil
}

type Bucket struct {
	Key      string `json:"key"`
	DocCount int64  `json:"doc_count"`
}
type Agg struct {
	Buckets []Bucket `json:"buckets"`
}
type AggResponse struct {
	Aggs map[string]Agg `json:"aggregations"`
}
