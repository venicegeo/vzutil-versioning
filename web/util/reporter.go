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

package util

import (
	"encoding/json"
	"errors"
	"sort"
	"strings"

	"github.com/venicegeo/pz-gocommon/elasticsearch"
	"github.com/venicegeo/vzutil-versioning/web/es"
	"github.com/venicegeo/vzutil-versioning/web/f"
)

type Reporter struct {
	index      *elasticsearch.Index
	searchSize int
}

func NewReporter(index *elasticsearch.Index) *Reporter {
	return &Reporter{index, 250}
}

func (r *Reporter) reportBySha(fullName, sha string) (res []es.Dependency, err error) {
	var project *es.Project
	var projectEntries *es.ProjectEntries
	var entry es.ProjectEntry
	var exists bool
	var resp *elasticsearch.GetResult

	if project, err = es.GetProjectById(r.index, fullName); err != nil {
		return nil, err
	}
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
	//TODO THREAD THIS NONSENSE
	for _, d := range entry.Dependencies {
		if resp, err = r.index.GetByID("dependency", d); err != nil || !resp.Found {
			name := f.Format("Cound not find [%s]", d)
			tmp := es.Dependency{name, "", ""}
			res = append(res, tmp)
		} else {
			var dep es.Dependency
			if err = json.Unmarshal([]byte(*resp.Source), &dep); err != nil {
				tmp := es.Dependency{f.Format("Error getting [%s]: [%s]", d, err.Error()), "", ""}
				res = append(res, tmp)
			} else {
				res = append(res, dep)
			}
		}
	}
	return res, nil
}

func (r *Reporter) reportByTag(info ...string) (map[string][]es.Dependency, error) {
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
	resp, err := es.MatchAllSize(r.index, "project", r.searchSize)
	if err != nil {
		return nil, err
	}
	hits := resp.GetHits()
	projects := []es.Project{}
	for _, hit := range *hits {
		var project es.Project
		if err = json.Unmarshal([]byte(*hit.Source), &project); err != nil {
			return nil, err
		}
		projects = append(projects, project)
	}

	res := map[string][]es.Dependency{}
	for _, project := range projects {
		tagShas, err := project.GetTagShas()
		if err != nil {
			return nil, err
		}
		sha, exists := (*tagShas)[tag]
		if !exists {
			continue
		}
		deps, err := r.reportBySha(project.FullName, sha)
		if err != nil {
			return nil, err
		}
		res[project.FullName] = deps
	}
	return res, nil
}

func (r *Reporter) reportByTag2(org, tag string) (map[string][]es.Dependency, error) {
	projectNames, err := r.listProjectsByOrg(org)
	if err != nil {
		return nil, err
	}
	res := map[string][]es.Dependency{}
	for _, name := range projectNames {
		project, err := es.GetProjectById(r.index, name)
		if err != nil {
			return nil, err
		}
		tagShas, err := project.GetTagShas()
		if err != nil {
			return nil, err
		}
		sha, exists := (*tagShas)[tag]
		if !exists {
			continue
		}
		deps, err := r.reportBySha(name, sha)
		if err != nil {
			return nil, err
		}
		res[name] = deps
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
	deps, err := r.reportBySha(docName, sha)
	if err != nil {
		return nil, err
	}
	return map[string][]es.Dependency{strings.Replace(docName, "_", "/", 1): deps}, nil
}

//

func (r *Reporter) listShas(fullName string) (res []string, err error) {
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

func (r *Reporter) listTagsRepo(fullName string) (*map[string]string, error) {
	project, err := es.GetProjectById(r.index, fullName)
	if err != nil {
		return nil, err
	}
	return project.GetTagShas()
}
func (r *Reporter) listTags(org string) (*map[string][]string, int, error) {
	resp, err := r.index.SearchByJSON("project", f.Format(`
{
	"size": %d,
	"query": {
		"regexp": {
			"full_name": "%s"
		}
	}
}	
	`, r.searchSize, org))
	if err != nil {
		return nil, 0, err
	}
	hits := resp.GetHits()
	mapp := map[string][]string{}
	var project es.Project
	numTags := 0
	for _, h := range *hits {
		if err = json.Unmarshal(*h.Source, &project); err != nil {
			return nil, 0, err
		}
		mapp[project.FullName] = []string{}
		tags, err := project.GetTagShas()
		if err != nil {
			return nil, 0, err
		}
		numTags += len(*tags)
		for tag, _ := range *tags {
			mapp[project.FullName] = append(mapp[project.FullName], tag)
		}
		sort.Strings(mapp[project.FullName])
	}
	return &mapp, numTags, err
}

//

func (r *Reporter) listProjects() ([]string, error) {
	return r.listProjectsWrk(es.MatchAllSize(r.index, "project", r.searchSize))

}
func (r *Reporter) listProjectsByOrg(org string) ([]string, error) {
	return r.listProjectsWrk(r.index.SearchByJSON("project", f.Format(`
{
	"size": %d,
	"query": {
		"regexp": {
			"full_name": "%s"
		}
	}
}	
	`, r.searchSize, org)))
}
func (r *Reporter) listProjectsWrk(resp *elasticsearch.SearchResult, err error) ([]string, error) {
	if err != nil {
		return nil, err
	}
	hits := *resp.GetHits()
	res := []string{}
	var project *es.Project
	for _, hit := range hits {
		if err = json.Unmarshal(*hit.Source, &project); err != nil {
			return nil, err
		}
		res = append(res, project.FullName)
	}
	sort.Strings(res)
	return res, nil
}
