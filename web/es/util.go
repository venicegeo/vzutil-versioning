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

type hit struct {
	Id  string
	Dat []byte
}

func GetAll(index *elasticsearch.Index, typ, query string) ([]*hit, error) {
	from := int64(0)
	size := int64(20)
	res := []*hit{}
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
			res = append(res, &hit{h.ID, *h.Source})
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
		if err = json.Unmarshal(dat.Dat, &dep); err != nil {
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

func MatchAllSize(index *elasticsearch.Index, typ string, size int) (*elasticsearch.SearchResult, error) {
	return index.SearchByJSON(typ, u.Format(`
{
	"size": %d,
	"query":{}
}	
	`, size))
}
