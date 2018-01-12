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
	"fmt"
	"strings"

	"github.com/venicegeo/pz-gocommon/elasticsearch"
	"github.com/venicegeo/vzutil-versioning/web/es"
)

type Reporter struct {
	index *elasticsearch.Index
}

func NewReporter(index *elasticsearch.Index) *Reporter {
	return &Reporter{index}
}

func (r *Reporter) ReportBySha(fullName, sha string) ([]string, error) {
	docName := strings.Replace(fullName, "/", "_", -1)
	resp, err := r.index.GetByID("project", docName)
	if err != nil {
		return nil, err
	}
	if !resp.Found {
		return nil, fmt.Errorf("Could not find a project named %s", docName)
	}
	var project es.Project
	if err = json.Unmarshal([]byte(*resp.Source), &project); err != nil {
		return nil, err
	}
	projectEntries, err := project.GetEntries()
	if err != nil {
		return nil, err
	}
	entry, ok := (*projectEntries)[sha]
	if !ok {
		return nil, fmt.Errorf("Sorry, this sha was not found")
	}
	if entry.EntryReference != "" {
		entry, ok = (*projectEntries)[entry.EntryReference]
		if !ok {
			return nil, fmt.Errorf("The database is corrupted, this sha points to a sha that doesnt exist:", entry.EntryReference)
		}
	}
	//TODO THREAD THIS NONSENSE
	res := []string{}
	for _, d := range entry.Dependencies {
		resp, err = r.index.GetByID("dependency", d)
		if err != nil || !resp.Found {
			res = append(res, fmt.Sprintf("Cound not find [%s]", d))
		} else {
			var dep es.Dependency
			if err = json.Unmarshal([]byte(*resp.Source), &dep); err != nil {
				res = append(res, fmt.Sprintf("Error getting [%s]: [%s]", d, err.Error()))
			} else {
				res = append(res, dep.String())
			}
		}
	}
	return res, nil
}

func (r *Reporter) ReportByTag(tag string) (map[string][]string, error) {
	resp, err := r.index.GetAllElements("project")
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

	mapp := map[string]string{}

	for _, project := range projects {
		tagShas, err := project.GetTagShas()
		if err != nil {
			return nil, err
		}
		sha, exists := tagShas[tag]
		if exists {
			mapp[project.FullName] = sha
		}
	}

	mappp := map[string][]string{}
	for projectName, sha := range mapp {
		deps, err := r.ReportBySha(projectName, sha)
		if err != nil {
			return nil, err
		}
		mappp[projectName] = deps
	}

	return mappp, nil
}
