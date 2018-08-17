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
	"github.com/venicegeo/pz-gocommon/elasticsearch/elastic-5-api"
)

func GetAll(index elasticsearch.IIndex, typ string, query interface{}, vsort ...interface{}) (*elastic.SearchHits, error) {
	return GetAllSource(index, typ, query, true, vsort...)
}
func GetAllSource(index elasticsearch.IIndex, typ string, query interface{}, source interface{}, vsort ...interface{}) (*elastic.SearchHits, error) {
	from := int64(0)
	size := int64(20)
	res := &elastic.SearchHits{0, nil, []*elastic.SearchHit{}}
	var sort interface{}
	if len(vsort) > 0 {
		sort = vsort[0]
	} else {
		sort = map[string]interface{}{}
	}
	for {
		q := map[string]interface{}{
			"from":    from,
			"size":    size,
			"_source": source,
			"query":   query,
			"sort":    sort,
		}
		result, err := index.SearchByJSON(typ, q)
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
