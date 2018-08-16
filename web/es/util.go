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

	"github.com/venicegeo/pz-gocommon/elasticsearch"
	"github.com/venicegeo/pz-gocommon/elasticsearch/elastic-5-api"
	u "github.com/venicegeo/vzutil-versioning/web/util"
)

func GetAll(index elasticsearch.IIndex, typ, query string, vsort ...string) (*elastic.SearchHits, error) {
	return GetAllSource(index, typ, query, true, vsort...)
}
func GetAllSource(index elasticsearch.IIndex, typ, query string, source interface{}, vsort ...string) (*elastic.SearchHits, error) {
	s, err := json.Marshal(source)
	if err != nil {
		return nil, err
	}
	from := int64(0)
	size := int64(20)
	res := &elastic.SearchHits{0, nil, []*elastic.SearchHit{}}
	sort := "{}"
	if len(vsort) > 0 {
		sort = vsort[0]
	}
	for {
		str := u.Format(`{
	"from":%d,
	"size":%d,
	"_source": %s,
	"query":%s,
	"sort":%s
}`, from, size, string(s), query, sort)
		result, err := index.SearchByJSON(typ, str)
		if err != nil {
			return nil, err
		}
		if len(result.Hits.Hits) == 0 {
			break
		}
		res.Hits = append(res.Hits, result.Hits.Hits...)
		if int64(len(res.Hits)) < size {
			break
		}
		from += size
	}
	res.TotalHits = int64(len(res.Hits))
	return res, nil
}
