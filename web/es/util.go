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
	"bytes"
	"encoding/json"
	"strings"
	"sync"

	"github.com/venicegeo/pz-gocommon/elasticsearch"
	u "github.com/venicegeo/vzutil-versioning/web/util"
)

func GetRepositoryById(index *elasticsearch.Index, fullName string) (*Repository, bool, error) {
	docName := strings.Replace(fullName, "/", "_", -1)
	resp, err := index.GetByID("repository", docName)
	if err != nil {
		return nil, false, err
	}
	if !resp.Found {
		return nil, false, nil
	}
	repo := &Repository{}
	d := json.NewDecoder(bytes.NewReader([]byte(*resp.Source)))
	d.UseNumber()
	if err = d.Decode(repo); err != nil {
		return nil, true, err
	}
	return repo, true, nil
}

func CheckShaExists(index *elasticsearch.Index, fullName string, sha string) (bool, error) {
	resp, err := index.SearchByJSON("repository", u.Format(`
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

func MatchAllSize(index *elasticsearch.Index, typ string, size int) (*elasticsearch.SearchResult, error) {
	return index.SearchByJSON(typ, u.Format(`
{
	"size": %d,
	"query":{}
}	
	`, size))
}

func GetAllRepositories(index *elasticsearch.Index, size int) (*[]*Repository, error) {
	return HitsToRepositories(MatchAllSize(index, "repository", size))
}

func GetRepositoriesOrg(index *elasticsearch.Index, org string, size int) (*[]*Repository, error) {
	return HitsToRepositories(index.SearchByJSON("repository", u.Format(`
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

func HitsToRepositories(resp *elasticsearch.SearchResult, err error) (*[]*Repository, error) {
	if err != nil {
		return nil, err
	}
	hits := *resp.GetHits()
	res := make([]*Repository, len(hits))
	mux := &sync.Mutex{}
	errs := make(chan error, len(hits))
	work := func(i int, hit *elasticsearch.SearchResultHit) {
		var repo Repository
		d := json.NewDecoder(bytes.NewReader(*hit.Source))
		d.UseNumber()
		if err = d.Decode(&repo); err != nil {
			errs <- err
			return
		}
		mux.Lock()
		res[i] = &repo
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
