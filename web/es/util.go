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
	"github.com/venicegeo/pz-gocommon/elasticsearch"
	u "github.com/venicegeo/vzutil-versioning/web/util"
)

type hit struct {
	Id  string
	Dat []byte
}

func GetAll(index *elasticsearch.Index, typ, query string, vsort ...string) ([]*hit, error) {
	from := int64(0)
	size := int64(20)
	res := []*hit{}
	sort := "{}"
	if len(vsort) > 0 {
		sort = vsort[0]
	}
	for {
		str := u.Format(`{
	"from":%d,
	"size":%d,
	"query":%s,
	"sort":%s
}`, from, size, query, sort)
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
