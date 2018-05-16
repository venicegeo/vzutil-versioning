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
	"sort"

	"github.com/venicegeo/pz-gocommon/elasticsearch"
	u "github.com/venicegeo/vzutil-versioning/web/util"
)

func GetAll(index *elasticsearch.Index, typ, query string) ([][]byte, error) {
	from := int64(0)
	size := int64(20)
	res := [][]byte{}
	for {
		str := u.Format(`{
	"from":%d,
	"size":%d,
	"query":%s
}`, from, size, query)
		result, err := index.SearchByJSON(typ, str)
		if err != nil {
			return nil, err
		}
		if !result.Found {
			return nil, u.Error("The %s type was not found", typ)
		}
		if result.NumHits() == 0 {
			break
		}
		from += size
		hits := result.GetHits()
		for _, h := range *hits {
			res = append(res, *h.Source)
		}
	}
	return res, nil
}

func GetAllDependencies(index *elasticsearch.Index) ([]Dependency, error) {
	depsDat, err := GetAll(index, "dependency", "{}")
	if err != nil {
		return nil, err
	}
	deps := make(DependencySort, len(depsDat), len(depsDat))
	for i, dat := range depsDat {
		var dep Dependency
		if err = json.Unmarshal(dat, &dep); err != nil {
			return nil, err
		}
		deps[i] = dep
	}
	sort.Sort(deps)
	return deps, nil
}
func GetAllDependenciesStr(index *elasticsearch.Index) ([]string, error) {
	deps, err := GetAllDependencies(index)
	if err != nil {
		return nil, err
	}
	res := make([]string, len(deps), len(deps))
	for i, dep := range deps {
		res[i] = dep.String()
	}
	return res, nil
}

//func GetRepositoryById(index *elasticsearch.Index, fullName string) (*Repository, bool, error) {
//	docName := strings.Replace(fullName, "/", "_", -1)
//	resp, err := index.GetByID("repository", docName)
//	if err != nil {
//		return nil, false, err
//	}
//	if !resp.Found {
//		return nil, false, nil
//	}
//	repo := &Repository{}
//	d := json.NewDecoder(bytes.NewReader([]byte(*resp.Source)))
//	d.UseNumber()
//	if err = d.Decode(repo); err != nil {
//		return nil, true, err
//	}
//	return repo, true, nil
//}

func MatchAllSize(index *elasticsearch.Index, typ string, size int) (*elasticsearch.SearchResult, error) {
	return index.SearchByJSON(typ, u.Format(`
{
	"size": %d,
	"query":{}
}	
	`, size))
}

//func GetAllRepositories(index *elasticsearch.Index, size int) (*[]*Repository, error) {
//	return HitsToRepositories(MatchAllSize(index, "repository", size))
//}

//func GetRepositoriesOrg(index *elasticsearch.Index, org string, size int) (*[]*Repository, error) {
//	return HitsToRepositories(index.SearchByJSON("repository", u.Format(`
//{
//	"size": %d,
//	"query": {
//		"wildcard": {
//			"full_name": "%s/*"
//		}
//	}
//}
//	`, size, org)))

//}

//func HitsToRepositories(resp *elasticsearch.SearchResult, err error) (*[]*Repository, error) {
//	if err != nil {
//		return nil, err
//	}
//	hits := *resp.GetHits()
//	res := make([]*Repository, len(hits))
//	mux := &sync.Mutex{}
//	errs := make(chan error, len(hits))
//	work := func(i int, hit *elasticsearch.SearchResultHit) {
//		var repo Repository
//		d := json.NewDecoder(bytes.NewReader(*hit.Source))
//		d.UseNumber()
//		if err = d.Decode(&repo); err != nil {
//			errs <- err
//			return
//		}
//		mux.Lock()
//		res[i] = &repo
//		mux.Unlock()
//		errs <- nil
//	}
//	for i, hit := range hits {
//		go work(i, hit)
//	}
//	for i := 0; i < len(hits); i++ {
//		err := <-errs
//		if err != nil {
//			return nil, err
//		}
//	}
//	return &res, nil
//}
