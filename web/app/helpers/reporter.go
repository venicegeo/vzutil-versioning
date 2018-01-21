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

func (r *Reporter) ReportByShaName(fullName, sha string) (res []es.Dependency, err error) {
	var project *es.Project

	if project, err = es.GetProjectById(r.index, fullName); err != nil {
		return nil, err
	}
	return r.ReportByShaProject(project, sha)

}
func (r *Reporter) ReportByShaProject(project *es.Project, sha string) (res []es.Dependency, err error) {
	var projectEntries *es.ProjectEntries
	var entry es.ProjectEntry
	var exists bool

	if projectEntries, err = project.GetEntries(); err != nil {
		return nil, err
	}
	entry, exists = (*projectEntries)[sha]
	if !exists {
		return nil, errors.New("Sorry, this sha was not found")
	}
	if entry.EntryReference != "" {
		entry, exists = (*projectEntries)[entry.EntryReference]
		if !exists {
			return nil, errors.New("The database is corrupted, this sha points to a sha that doesnt exist: " + entry.EntryReference)
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
	return res, nil
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
		tagShas, err := project.GetTagShas()
		if err != nil {
			errs <- err
			return
		}
		sha, exists := (*tagShas)[tag]
		if !exists {
			errs <- err
			return
		}
		deps, err := r.ReportByShaProject(project, sha)
		if err != nil {
			errs <- err
			return
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
		tagShas, err := project.GetTagShas()
		if err != nil {
			errs <- err
			return
		}
		sha, exists := (*tagShas)[tag]
		if !exists {
			errs <- nil
			return
		}
		deps, err := r.ReportByShaProject(project, sha)
		if err != nil {
			errs <- err
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
	var tagShas *map[string]string
	var sha string
	var ok bool

	if project, err = es.GetProjectById(r.index, docName); err != nil {
		return nil, err
	}
	if tagShas, err = project.GetTagShas(); err != nil {
		return nil, err
	}
	if sha, ok = (*tagShas)[tag]; !ok {
		return nil, errors.New("Could not find this tag: [" + tag + "]")
	}
	deps, err := r.ReportByShaProject(project, sha)
	if err != nil {
		return nil, err
	}
	return map[string][]es.Dependency{strings.Replace(docName, "_", "/", 1): deps}, nil
}

//

func (r *Reporter) ListShas(fullName string) (res []string, err error) {
	var project *es.Project
	var entries *es.ProjectEntries

	if project, err = es.GetProjectById(r.index, fullName); err != nil {
		return nil, err
	}
	if entries, err = project.GetEntries(); err != nil {
		return nil, err
	}
	for k, _ := range *entries {
		res = append(res, k)
	}
	sort.Strings(res)
	return res, nil
}

//

func (r *Reporter) ListTagsRepo(fullName string) (*map[string]string, error) {
	project, err := es.GetProjectById(r.index, fullName)
	if err != nil {
		return nil, err
	}
	return project.GetTagShas()
}
func (r *Reporter) ListTags(org string) (*map[string][]string, int, error) {
	projects, err := es.GetProjectsOrg(r.index, org, r.searchSize)
	if err != nil {
		return nil, 0, err
	}
	res := map[string][]string{}
	numTags := 0
	errs := make(chan error, len(*projects))
	mux := &sync.Mutex{}

	work := func(project *es.Project) {
		temp := []string{}
		tags, err := project.GetTagShas()
		if err != nil {
			errs <- err
			return
		}
		for tag, _ := range *tags {
			temp = append(temp, tag)
		}
		sort.Strings(temp)
		mux.Lock()
		{
			numTags += len(*tags)
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
	res := []string{}
	for _, project := range *projects {
		res = append(res, project.FullName)
	}
	sort.Strings(res)
	return res, nil
}
