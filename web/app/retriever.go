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
		fmt.Println("what")
		res := map[string][]es.Dependency{}
		res["test"] = []es.Dependency{es.Dependency{"a", "b", "c"}}

		in := r.newAggQuery("repos", "repo_fullname")
		in["aggs"].(map[string]interface{})["repos"].(map[string]interface{})["aggs"] = map[string]interface{}{
			"max_time": map[string]interface{}{
				"max": map[string]interface{}{
					"field": "timestamp",
				},
			},
		}
		in["query"] = map[string]interface{}{
			"term": map[string]interface{}{
				"ref_name": "refs/" + info[0],
			},
		}
		dat, _ := json.MarshalIndent(in, " ", "   ")
		fmt.Println(string(dat))
		var out AggResponse
		//TODO
		if err := r.app.index.DirectAccess("GET", "/versioning_tool/repository_entry/_search", in, &out); err != nil {
			return nil, err
		}
		for _, bucket := range out.Aggs["repos"].Buckets {
			repoName := bucket.Key
			time := bucket.MaxTime
			fmt.Println(repoName, time)
		}

		return res, nil
	//		ref := info[0]
	//	re, e := es.GetAllRepositories(r.app.index, r.app.searchSize)
	//	return r.byRefWork(re, e, ref)
	case 2: //org and ref
	//	org := info[0]
	//	ref := info[1]
	//	re, e := es.GetRepositoriesOrg(r.app.index, org, r.app.searchSize)
	//	return r.byRefWork(re, e, ref)
	case 3: //org repo and ref
		//	org := info[0]
		//	repo := info[1]
		//	ref := info[2]
		//	re, o, e := es.GetRepositoryById(r.app.index, org+"_"+repo)
		//	if !o {
		//		return nil, u.Error("Unable to find doc [%s_%s]", org, repo)
		//	}
		//	return r.byRefWork(&[]*es.Repository{re}, e, ref)
	}
	return nil, u.Error("Sorry, something is wrong with the code..")
}

//func (r *Retriever) byRefWork(repos *[]*es.Repository, err error, ref string) (map[string][]es.Dependency, error) {
//	if err != nil {
//		return nil, err
//	}
//	res := map[string][]es.Dependency{}
//	mux := &sync.Mutex{}
//	errs := make(chan error, len(*repos))
//	work := func(repo *es.Repository, e chan error) {
//		var refp *es.Ref = nil
//		for _, r := range repo.Refs {
//			if r.Name == `refs/`+ref {
//				refp = r
//				break
//			}
//		}
//		if refp == nil {
//			e <- nil
//			return
//		}
//		if len(refp.WebhookOrder) == 0 {
//			e <- nil
//			return
//		}
//		sha := refp.WebhookOrder[0]
//		deps, found, err := r.DepsByShaRepository(repo, sha)
//		if err != nil {
//			e <- err
//			return
//		} else if !found {
//			e <- u.Error("Could not find sha [%s]", sha)
//			return
//		}
//		sort.Sort(es.DependencySort(deps))
//		mux.Lock()
//		{
//			res[repo.FullName] = deps
//		}
//		mux.Unlock()
//		e <- nil
//	}
//	for _, repo := range *repos {
//		go work(repo, errs)
//	}
//	for i := 0; i < len(*repos); i++ {
//		err := <-errs
//		if err != nil {
//			return nil, err
//		}
//	}
//	return res, nil
//}

//func (r *Retriever) byRef1(ref string) (map[string][]es.Dependency, error) {
//	repos, err := es.GetAllRepositories(r.app.index, r.app.searchSize)
//	if err != nil {
//		return nil, err
//	}

//	res := map[string][]es.Dependency{}
//	mux := &sync.Mutex{}
//	errs := make(chan error, len(*repos))
//	work := func(repo *es.Repository) {
//		var refp *es.Ref = nil
//		for _, r := range repo.Refs {
//			if r.Name == ref {
//				refp = r
//				break
//			}
//		}
//		if refp == nil {
//			return
//		}
//		if len(refp.WebhookOrder) == 0 {
//			return
//		}
//		sha := refp.WebhookOrder[0]
//		deps, found, err := r.DepsByShaRepository(repo, sha)
//		if err != nil {
//			errs <- err
//			return
//		} else if !found {
//			errs <- u.Error("Could not find sha [%s]", sha)
//		}
//		mux.Lock()
//		{
//			res[repo.FullName] = deps
//			errs <- nil
//		}
//		mux.Unlock()
//	}
//	for _, repo := range *repos {
//		go work(repo)
//	}
//	for i := 0; i < len(*repos); i++ {
//		err := <-errs
//		if err != nil {
//			return nil, err
//		}
//	}
//	return res, nil
//}

//func (r *Retriever) byRef2(org, tag string) (map[string][]es.Dependency, error) {
//	repos, err := es.GetRepositoriesOrg(r.app.index, org, r.app.searchSize)
//	if err != nil {
//		return nil, err
//	}
//	res := map[string][]es.Dependency{}
//	mux := &sync.Mutex{}
//	errs := make(chan error, len(*repos))
//	work := func(repo *es.Repository) {
//		sha, exists := repo.GetTagFromSha(tag)
//		if !exists {
//			errs <- errors.New("Could not find sha for tag " + tag)
//			return
//		}
//		deps, found, err := r.DepsByShaRepository(repo, sha)
//		if err != nil {
//			errs <- err
//			return
//		} else if !found {
//			errs <- u.Error("Repository [%s] not found", repo.FullName)
//			return
//		}
//		mux.Lock()
//		res[repo.FullName] = deps
//		mux.Unlock()
//		errs <- nil
//	}
//	for _, repo := range *repos {
//		go work(repo)
//	}
//	err = nil
//	for i := 0; i < len(*repos); i++ {
//		e := <-errs
//		if e != nil {
//			err = e
//		}
//	}
//	return res, err
//}

//func (r *Retriever) byRef3(docName, tag string) (map[string][]es.Dependency, error) {
//	var repo *es.Repository
//	var err error
//	var sha string
//	var ok bool

//	if repo, ok, err = es.GetRepositoryById(r.app.index, docName); err != nil {
//		return nil, err
//	} else if !ok {
//		return nil, u.Error("Could not find [%s]", docName)
//	}
//	ok = false

//	if sha, ok = repo.GetShaFromTag(tag); !ok {
//		return nil, errors.New("Could not find this tag: [" + tag + "]")
//	}
//	deps, found, err := r.DepsByShaRepository(repo, sha)
//	if err != nil {
//		return nil, err
//	} else if !found {
//		return nil, u.Error("Could not find the dependencies for sha [%s] on repository [%s]", sha, docName)
//	}
//	return map[string][]es.Dependency{strings.Replace(docName, "_", "/", 1): deps}, nil
//}

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
		if err := json.Unmarshal(entryD, &entry); err != nil {
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

type MaxInt64 struct {
	Value float64 `json:"value"`
}
type Bucket struct {
	Key      string   `json:"key"`
	DocCount int64    `json:"doc_count"`
	MaxTime  MaxInt64 `json:"max_time_int64"`
}
type Agg struct {
	Buckets []Bucket `json:"buckets"`
}
type AggResponse struct {
	Aggs map[string]Agg `json:"aggregations"`
}
