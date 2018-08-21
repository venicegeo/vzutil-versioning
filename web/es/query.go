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
	"errors"

	"github.com/venicegeo/pz-gocommon/elasticsearch/elastic-5-api"
)

type Bool struct {
	Must    *BoolQ `json:"must,omitempty"`
	Should  *BoolQ `json:"should,omitempty"`
	MustNot *BoolQ `json:"must_not,omitempty"`
	Filter  *BoolQ `json:"filter,omitempty"`
}
type BoolQ []interface{}
type Term struct {
	Term map[string]string `json:"term"`
}
type Wildcard struct {
	Wildcard map[string]string `json:"wildcard"`
}
type Terms struct {
	Terms map[string][]string `json:"terms"`
}
type Range struct {
	Range map[string]map[string]interface{} `json:"range"`
}

func NewTerm(key, value string) *Term {
	ret := new(Term)
	ret.Term = map[string]string{key: value}
	return ret
}
func NewWildcard(key, value string) *Wildcard {
	ret := new(Wildcard)
	ret.Wildcard = map[string]string{key: value}
	return ret
}
func NewTerms(key string, value ...string) *Terms {
	ret := new(Terms)
	ret.Terms = map[string][]string{key: value}
	return ret
}
func NewRange(key, typ string, value interface{}) *Range {
	ret := new(Range)
	ret.Range = map[string]map[string]interface{}{key: map[string]interface{}{typ: value}}
	return ret
}
func NewBoolQ(items ...interface{}) *BoolQ {
	ret := make(BoolQ, len(items), len(items))
	for i, it := range items {
		ret[i] = it
	}
	return &ret
}
func (bq *BoolQ) Add(item interface{}) *BoolQ {
	*bq = append(*bq, item)
	return bq
}
func NewBool() *Bool {
	return new(Bool)
}
func (b *Bool) SetMust(bq *BoolQ) *Bool {
	b.Must = bq
	return b
}
func (b *Bool) SetShould(bq *BoolQ) *Bool {
	b.Should = bq
	return b
}
func (b *Bool) SetMustNot(bq *BoolQ) *Bool {
	b.MustNot = bq
	return b
}
func (b *Bool) SetFilter(bq *BoolQ) *Bool {
	b.Filter = bq
	return b
}
func NewAggQuery(aggName, fieldName string) map[string]interface{} {
	return map[string]interface{}{
		"aggs": map[string]interface{}{
			aggName: map[string]interface{}{
				"terms": map[string]interface{}{
					"field": fieldName,
					"size":  10000,
				},
			},
		},
		"size": 0}
}

//type Bucket struct {
//	Key      string `json:"key"`
//	DocCount int64  `json:"doc_count"`
//}
//type Agg struct {
//	Buckets []Bucket `json:"buckets"`
//}

//func (a Agg) GetKeys() []string {
//	res := make([]string, len(a.Buckets), len(a.Buckets))
//	for i, b := range a.Buckets {
//		res[i] = b.Key
//	}
//	return res
//}

//type AggResponse struct {
//	Aggs map[string]Agg `json:"aggregations"`
//}

func GetAggKeysFromSearchResponse(term string, resp *elastic.SearchResult, err error, transform ...func(string) string) ([]string, error) {
	if err != nil {
		return nil, err
	}
	var trans func(string) string
	if len(transform) > 0 {
		trans = transform[0]
	} else {
		trans = func(a string) string {
			return a
		}
	}
	agg, ok := resp.Aggregations.Terms(term)
	if !ok {
		return nil, errors.New("Agg query failed to find agg term")
	}
	res := make([]string, len(agg.Buckets), len(agg.Buckets))
	for i, b := range agg.Buckets {
		res[i] = trans(b.Key.(string))
	}
	return res, nil
}

type NestedQuery map[string]interface{}

func NewNestedQuery(path string) NestedQuery {
	return map[string]interface{}{
		"nested": map[string]interface{}{
			"path":       path,
			"inner_hits": map[string]interface{}{},
		},
	}
}
func (n *NestedQuery) SetInnerQuery(query interface{}) map[string]interface{} {
	(*n)["nested"].(map[string]interface{})["query"] = query
	return *n
}
