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

package es

import (
	"encoding/json"
	"strings"
	"sync"

	"github.com/venicegeo/pz-gocommon/elasticsearch"
	u "github.com/venicegeo/vzutil-versioning/web/util"
)

func GetProjectById(index *elasticsearch.Index, fullName string) (*Project, bool, error) {
	docName := strings.Replace(fullName, "/", "_", -1)
	resp, err := index.GetByID("project", docName)
	if err != nil {
		return nil, false, err
	}
	if !resp.Found {
		return nil, false, nil
	}
	project := &Project{}
	if err = json.Unmarshal([]byte(*resp.Source), project); err != nil {
		return nil, true, err
	}
	return project, true, nil
}

func CheckShaExists(index *elasticsearch.Index, fullName string, sha string) (bool, error) {
	resp, err := index.SearchByJSON("project", u.Format(`
{
	"query": {
		"bool":{
			"must":[{
				"term":{
					"full_name":"%s"
				}	
			},{
				"term": {
					"refs.entries.sha": "%s"
				}
			}]
		}
	}
}`, fullName, sha))
	if err != nil {
		return false, err
	}
	return resp.NumHits() > 0, nil
}

func GetShaFromTag(index *elasticsearch.Index, fullName, tag string) (string, bool, error) {
	resp, err := index.SearchByJSON("project", u.Format(`
{
	"query": {
		"bool":{
			"must":[{
				"term":{
					"full_name":"%s"
				}	
			},{
				"term": {
					"tag_shas.tag": "%s"
				}
			}]
		}
	}
}`, fullName, tag))
	if err != nil {
		return "", false, err
	}
	if resp.NumHits() <= 0 {
		return "", false, nil
	}
	var ts []TagSha
	if err = json.Unmarshal([]byte(*resp.GetHit(0).Source), &ts); err != nil {
		return "", false, err
	}
	for _, e := range ts {
		if e.Tag == tag {
			return e.Sha, true, nil
		}
	}
	return "", false, nil
}

func MatchAllSize(index *elasticsearch.Index, typ string, size int) (*elasticsearch.SearchResult, error) {
	return index.SearchByJSON(typ, u.Format(`
{
	"size": %d,
	"query":{}
}	
	`, size))
}

func GetAllProjects(index *elasticsearch.Index, size int) (*[]*Project, error) {
	return HitsToProjects(MatchAllSize(index, "project", size))
}

func GetProjectsOrg(index *elasticsearch.Index, org string, size int) (*[]*Project, error) {
	return HitsToProjects(index.SearchByJSON("project", u.Format(`
{
	"size": %d,
	"query": {
		"wildcard": {
			"full_name": "%s/*"
		}
	}
}	
	`, size, org)))

}

func HitsToProjects(resp *elasticsearch.SearchResult, err error) (*[]*Project, error) {
	if err != nil {
		return nil, err
	}
	hits := *resp.GetHits()
	res := make([]*Project, len(hits))
	mux := &sync.Mutex{}
	errs := make(chan error, len(hits))
	work := func(i int, hit *elasticsearch.SearchResultHit) {
		var project Project
		if err = json.Unmarshal(*hit.Source, &project); err != nil {
			errs <- err
			return
		}
		mux.Lock()
		res[i] = &project
		mux.Unlock()
		errs <- nil
	}
	for i, hit := range hits {
		go work(i, hit)
	}
	for i := 0; i < len(hits); i++ {
		err := <-errs
		if err != nil {
			return nil, err
		}
	}
	return &res, nil
}
