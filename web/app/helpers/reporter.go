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

package helpers

import (
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"sync"

	"github.com/venicegeo/pz-gocommon/elasticsearch"
	"github.com/venicegeo/vzutil-versioning/web/es"
	u "github.com/venicegeo/vzutil-versioning/web/util"
)

type Reporter struct {
	index      *elasticsearch.Index
	searchSize int
}

func NewReporter(index *elasticsearch.Index) *Reporter {
	return &Reporter{index, 250}
}

func (r *Reporter) ReportByShaName(fullName, sha string) ([]es.Dependency, bool, error) {
	var project *es.Project
	var err error
	var found bool

	if project, found, err = es.GetProjectById(r.index, fullName); err != nil {
		return nil, found, err
	} else if !found {
		return nil, false, nil
	}
	return r.ReportByShaProject(project, sha)

}
func (r *Reporter) ReportByShaProject(project *es.Project, sha string) (res []es.Dependency, exists bool, err error) {
	ref, entry, exists := project.GetEntry(sha)
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
		if resp, err := r.index.GetByID("dependency", dep); err != nil || !resp.Found {
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
	return res, true, nil
}

func (r *Reporter) ReportByTag(info ...string) (map[string][]es.Dependency, error) {
	switch len(info) {
	case 1: //just a tag
		tag := info[0]
		return r.reportByTag1(tag)
	case 2: //org and tag
		org := info[0]
		tag := info[1]
		return r.reportByTag2(org, tag)
	case 3: //org repo and tag
		org := info[0]
		repo := info[1]
		tag := info[2]
		return r.reportByTag3(tag, org+"_"+repo)
	}
	return nil, errors.New("Sorry, something is wrong with the code..")
}

func (r *Reporter) reportByTag1(tag string) (map[string][]es.Dependency, error) {
	projects, err := es.GetAllProjects(r.index, r.searchSize)
	if err != nil {
		return nil, err
	}

	res := map[string][]es.Dependency{}
	mux := &sync.Mutex{}
	errs := make(chan error, len(*projects))
	work := func(project *es.Project) {
		sha, exists := project.GetShaFromTag(tag)
		if !exists {
			errs <- errors.New("Could not find a sha for tag " + tag)
			return
		}
		deps, found, err := r.ReportByShaProject(project, sha)
		if err != nil {
			errs <- err
			return
		} else if !found {

		}
		mux.Lock()
		{
			res[project.FullName] = deps
			errs <- nil
		}
		mux.Unlock()
	}
	for _, project := range *projects {
		go work(project)
	}
	for i := 0; i < len(*projects); i++ {
		err := <-errs
		if err != nil {
			return nil, err
		}
	}
	return res, nil
}

func (r *Reporter) reportByTag2(org, tag string) (map[string][]es.Dependency, error) {
	projects, err := es.GetProjectsOrg(r.index, org, r.searchSize)
	if err != nil {
		return nil, err
	}
	res := map[string][]es.Dependency{}
	mux := &sync.Mutex{}
	errs := make(chan error, len(*projects))
	work := func(project *es.Project) {
		sha, exists := project.GetTagFromSha(tag)
		if !exists {
			errs <- errors.New("Could not find sha for tag " + tag)
			return
		}
		deps, found, err := r.ReportByShaProject(project, sha)
		if err != nil {
			errs <- err
			return
		} else if !found {
			errs <- u.Error("Project [%s] not found", project.FullName)
			return
		}
		mux.Lock()
		res[project.FullName] = deps
		mux.Unlock()
		errs <- nil
	}
	for _, project := range *projects {
		go work(project)
	}
	for i := 0; i < len(*projects); i++ {
		err := <-errs
		if err != nil {
			return nil, err
		}
	}
	return res, nil
}

func (r *Reporter) reportByTag3(tag, docName string) (map[string][]es.Dependency, error) {
	var project *es.Project
	var err error
	var sha string
	var ok bool

	if project, ok, err = es.GetProjectById(r.index, docName); err != nil {
		return nil, err
	} else if !ok {
		return nil, u.Error("Could not find [%s]", docName)
	}
	ok = false

	for _, ts := range project.TagShas {
		if ts.Tag == tag {
			sha = ts.Sha
			ok = true
			break
		}
	}
	if !ok {
		return nil, errors.New("Could not find this tag: [" + tag + "]")
	}
	deps, found, err := r.ReportByShaProject(project, sha)
	if err != nil {
		return nil, err
	} else if !found {
		return nil, u.Error("Could not find p")
	}
	return map[string][]es.Dependency{strings.Replace(docName, "_", "/", 1): deps}, nil
}

//

func (r *Reporter) ListShas(fullName string) (map[string][]string, int, error) {
	var project *es.Project
	res := map[string][]string{}
	count := 0
	var err error
	var found bool

	if project, found, err = es.GetProjectById(r.index, fullName); err != nil {
		return nil, 0, err
	} else if !found {
		return nil, 0, u.Error("Could not find project [%s]", fullName)
	}

	for _, ref := range project.Refs {
		res[ref.Name] = make([]string, len(ref.Entries), len(ref.Entries))
		for i, s := range ref.Entries {
			res[ref.Name][i] = s.Sha
		}
		count += len(ref.Entries)
	}
	return res, count, nil
}

//

func (r *Reporter) ListRefsRepo(fullName string) (*[]string, error) {
	project, found, err := es.GetProjectById(r.index, fullName)
	if err != nil {
		return nil, err
	} else if !found {
		return nil, u.Error("Could not find project [%s]", fullName)
	}
	res := make([]string, len(project.Refs), len(project.Refs))
	for i, r := range project.Refs {
		res[i] = strings.TrimPrefix(r.Name, `refs/`)
	}
	return &res, nil
}
func (r *Reporter) ListRefs(org string) (*map[string][]string, int, error) {
	projects, err := es.GetProjectsOrg(r.index, org, r.searchSize)
	if err != nil {
		return nil, 0, err
	}
	res := map[string][]string{}
	numTags := 0
	errs := make(chan error, len(*projects))
	mux := &sync.Mutex{}

	work := func(project *es.Project) {
		num := len(project.Refs)
		temp := make([]string, num, num)
		for i, r := range project.Refs {
			temp[i] = strings.TrimPrefix(r.Name, `refs/`)
		}
		sort.Strings(temp)
		mux.Lock()
		{
			numTags += num
			res[project.FullName] = temp
			errs <- nil
		}
		mux.Unlock()
	}

	for _, p := range *projects {
		go work(p)
	}
	for i := 0; i < len(*projects); i++ {
		err := <-errs
		if err != nil {
			return nil, 0, err
		}
	}
	return &res, numTags, err
}

//

func (r *Reporter) ListProjects() ([]string, error) {
	return r.listProjectsWrk(es.GetAllProjects(r.index, r.searchSize))

}
func (r *Reporter) ListProjectsByOrg(org string) ([]string, error) {
	return r.listProjectsWrk(es.GetProjectsOrg(r.index, org, r.searchSize))
}
func (r *Reporter) listProjectsWrk(projects *[]*es.Project, err error) ([]string, error) {
	if err != nil {
		return nil, err
	}
	res := make([]string, len(*projects))
	for i, project := range *projects {
		res[i] = project.FullName
	}
	sort.Strings(res)
	return res, nil
}
