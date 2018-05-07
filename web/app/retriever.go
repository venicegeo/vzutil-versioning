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
	"errors"
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

func (r *Retriever) DepsByShaName(fullName, sha string) ([]es.Dependency, bool, error) {
	var repo *es.Repository
	var err error
	var found bool

	if repo, found, err = es.GetRepositoryById(r.app.index, fullName); err != nil {
		return nil, found, err
	} else if !found {
		return nil, false, nil
	}
	return r.DepsByShaRepository(repo, sha)
}
func (r *Retriever) DepsByShaNameGen(fullName, sha string) ([]es.Dependency, error) {
	deps, found, err := r.app.rtrvr.DepsByShaName(fullName, sha)
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
func (r *Retriever) DepsByShaRepository(repo *es.Repository, sha string) (res []es.Dependency, exists bool, err error) {
	ref, entry, exists := repo.GetEntry(sha)
	if !exists {
		return nil, false, nil
	}
	if entry.EntryReference != "" {
		if entry, exists = ref.GetEntry(entry.EntryReference); !exists {
			return nil, true, errors.New("The database is corrupted, this sha points to a sha that doesnt exist: " + entry.EntryReference)
		}
	}
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
	return res, true, nil
}

func (r *Retriever) DepsByRef(info ...string) (map[string][]es.Dependency, error) {
	switch len(info) {
	case 1: //just a ref
		ref := info[0]
		re, e := es.GetAllRepositories(r.app.index, r.app.searchSize)
		return r.byRefWork(re, e, ref)
	case 2: //org and ref
		org := info[0]
		ref := info[1]
		re, e := es.GetRepositoriesOrg(r.app.index, org, r.app.searchSize)
		return r.byRefWork(re, e, ref)
	case 3: //org repo and ref
		org := info[0]
		repo := info[1]
		ref := info[2]
		re, o, e := es.GetRepositoryById(r.app.index, org+"_"+repo)
		if !o {
			return nil, u.Error("Unable to find doc [%s_%s]", org, repo)
		}
		return r.byRefWork(&[]*es.Repository{re}, e, ref)
	}
	return nil, errors.New("Sorry, something is wrong with the code..")
}

func (r *Retriever) byRefWork(repos *[]*es.Repository, err error, ref string) (map[string][]es.Dependency, error) {
	if err != nil {
		return nil, err
	}
	res := map[string][]es.Dependency{}
	mux := &sync.Mutex{}
	errs := make(chan error, len(*repos))
	work := func(repo *es.Repository, e chan error) {
		var refp *es.Ref = nil
		for _, r := range repo.Refs {
			if r.Name == `refs/`+ref {
				refp = r
				break
			}
		}
		if refp == nil {
			e <- nil
			return
		}
		if len(refp.WebhookOrder) == 0 {
			e <- nil
			return
		}
		sha := refp.WebhookOrder[0]
		deps, found, err := r.DepsByShaRepository(repo, sha)
		if err != nil {
			e <- err
			return
		} else if !found {
			e <- u.Error("Could not find sha [%s]", sha)
			return
		}
		sort.Sort(es.DependencySort(deps))
		mux.Lock()
		{
			res[repo.FullName] = deps
		}
		mux.Unlock()
		e <- nil
	}
	for _, repo := range *repos {
		go work(repo, errs)
	}
	for i := 0; i < len(*repos); i++ {
		err := <-errs
		if err != nil {
			return nil, err
		}
	}
	return res, nil
}

func (r *Retriever) byRef1(ref string) (map[string][]es.Dependency, error) {
	repos, err := es.GetAllRepositories(r.app.index, r.app.searchSize)
	if err != nil {
		return nil, err
	}

	res := map[string][]es.Dependency{}
	mux := &sync.Mutex{}
	errs := make(chan error, len(*repos))
	work := func(repo *es.Repository) {
		var refp *es.Ref = nil
		for _, r := range repo.Refs {
			if r.Name == ref {
				refp = r
				break
			}
		}
		if refp == nil {
			return
		}
		if len(refp.WebhookOrder) == 0 {
			return
		}
		sha := refp.WebhookOrder[0]
		deps, found, err := r.DepsByShaRepository(repo, sha)
		if err != nil {
			errs <- err
			return
		} else if !found {
			errs <- u.Error("Could not find sha [%s]", sha)
		}
		mux.Lock()
		{
			res[repo.FullName] = deps
			errs <- nil
		}
		mux.Unlock()
	}
	for _, repo := range *repos {
		go work(repo)
	}
	for i := 0; i < len(*repos); i++ {
		err := <-errs
		if err != nil {
			return nil, err
		}
	}
	return res, nil
}

func (r *Retriever) byRef2(org, tag string) (map[string][]es.Dependency, error) {
	repos, err := es.GetRepositoriesOrg(r.app.index, org, r.app.searchSize)
	if err != nil {
		return nil, err
	}
	res := map[string][]es.Dependency{}
	mux := &sync.Mutex{}
	errs := make(chan error, len(*repos))
	work := func(repo *es.Repository) {
		sha, exists := repo.GetTagFromSha(tag)
		if !exists {
			errs <- errors.New("Could not find sha for tag " + tag)
			return
		}
		deps, found, err := r.DepsByShaRepository(repo, sha)
		if err != nil {
			errs <- err
			return
		} else if !found {
			errs <- u.Error("Repository [%s] not found", repo.FullName)
			return
		}
		mux.Lock()
		res[repo.FullName] = deps
		mux.Unlock()
		errs <- nil
	}
	for _, repo := range *repos {
		go work(repo)
	}
	err = nil
	for i := 0; i < len(*repos); i++ {
		e := <-errs
		if e != nil {
			err = e
		}
	}
	return res, err
}

func (r *Retriever) byRef3(docName, tag string) (map[string][]es.Dependency, error) {
	var repo *es.Repository
	var err error
	var sha string
	var ok bool

	if repo, ok, err = es.GetRepositoryById(r.app.index, docName); err != nil {
		return nil, err
	} else if !ok {
		return nil, u.Error("Could not find [%s]", docName)
	}
	ok = false

	for _, ts := range repo.TagShas {
		if ts.Tag == tag {
			sha = ts.Sha
			ok = true
			break
		}
	}
	if !ok {
		return nil, errors.New("Could not find this tag: [" + tag + "]")
	}
	deps, found, err := r.DepsByShaRepository(repo, sha)
	if err != nil {
		return nil, err
	} else if !found {
		return nil, u.Error("Could not find the dependencies for sha [%s] on repository [%s]", sha, docName)
	}
	return map[string][]es.Dependency{strings.Replace(docName, "_", "/", 1): deps}, nil
}

//

func (r *Retriever) ListShas(fullName string) (map[string][]string, int, error) {
	var repo *es.Repository
	res := map[string][]string{}
	count := 0
	var err error
	var found bool

	if repo, found, err = es.GetRepositoryById(r.app.index, fullName); err != nil {
		return nil, 0, err
	} else if !found {
		return nil, 0, u.Error("Could not find repository [%s]", fullName)
	}

	for _, ref := range repo.Refs {
		res[ref.Name] = make([]string, len(ref.Entries), len(ref.Entries))
		for i, s := range ref.Entries {
			res[ref.Name][i] = s.Sha
		}
		count += len(ref.Entries)
	}
	return res, count, nil
}

//

func (r *Retriever) ListRefsRepo(fullName string) (*[]string, error) {
	repo, found, err := es.GetRepositoryById(r.app.index, fullName)
	if err != nil {
		return nil, err
	} else if !found {
		return nil, u.Error("Could not find repository [%s]", fullName)
	}
	res := make([]string, len(repo.Refs), len(repo.Refs))
	for i, r := range repo.Refs {
		res[i] = strings.TrimPrefix(r.Name, `refs/`)
	}
	return &res, nil
}
func (r *Retriever) ListRefs(org string) (*map[string][]string, int, error) {
	repos, err := es.GetRepositoriesOrg(r.app.index, org, r.app.searchSize)
	if err != nil {
		return nil, 0, err
	}
	res := map[string][]string{}
	numTags := 0
	errs := make(chan error, len(*repos))
	mux := &sync.Mutex{}

	work := func(repo *es.Repository) {
		num := len(repo.Refs)
		temp := make([]string, num, num)
		for i, r := range repo.Refs {
			temp[i] = strings.TrimPrefix(r.Name, `refs/`)
		}
		sort.Strings(temp)
		mux.Lock()
		{
			numTags += num
			res[repo.FullName] = temp
			errs <- nil
		}
		mux.Unlock()
	}

	for _, p := range *repos {
		go work(p)
	}
	for i := 0; i < len(*repos); i++ {
		err := <-errs
		if err != nil {
			return nil, 0, err
		}
	}
	return &res, numTags, err
}

//

func (r *Retriever) ListRepositories() ([]string, error) {
	return r.listRepositoriesWrk(es.GetAllRepositories(r.app.index, r.app.searchSize))

}
func (r *Retriever) ListRepositoriesByOrg(org string) ([]string, error) {
	return r.listRepositoriesWrk(es.GetRepositoriesOrg(r.app.index, org, r.app.searchSize))
}
func (r *Retriever) listRepositoriesWrk(repos *[]*es.Repository, err error) ([]string, error) {
	if err != nil {
		return nil, err
	}
	res := make([]string, len(*repos))
	for i, repo := range *repos {
		res[i] = repo.FullName
	}
	sort.Strings(res)
	return res, nil
}
