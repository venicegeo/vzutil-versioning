// Copyright 2016, RadiantBlue Technologies, Inc.
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

package elasticsearch

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"

	"github.com/venicegeo/pz-gocommon/elasticsearch/elastic-5-api"
	"github.com/venicegeo/pz-gocommon/gocommon"
)

const percolateTypeName = ".percolate"

type AnyType struct {
	Id   string
	S    map[string]interface{}
	vars map[string]interface{}
	raw  *json.RawMessage
}

func newAnyType(id string, i interface{}) (*AnyType, error) {
	if id == "" {
		id = piazza.NewUuid().String()
	}
	dat, err := json.Marshal(i)
	if err != nil {
		return nil, err
	}
	var s map[string]interface{}
	if err = json.Unmarshal(dat, &s); err != nil {
		return nil, err
	}
	raw := new(json.RawMessage)
	if err = raw.UnmarshalJSON(dat); err != nil {
		return nil, err
	}
	ret := new(AnyType)
	ret.S = s
	if ret.vars, err = piazza.GetVarsFromStruct(s); err != nil {
		return nil, err
	}
	ret.raw = raw
	ret.Id = id
	return ret, nil
}

type MockIndexType struct {
	// maps from id string to document body
	items map[string]*AnyType

	mapping interface{}
}

var _ IIndex = (*MockIndex)(nil)

type MockIndex struct {
	name     string
	types    map[string]*MockIndexType
	exists   bool
	open     bool
	settings interface{}
	idSource int
}

func NewMockIndex(indexName string) *MockIndex {
	var _ IIndex = new(MockIndex)

	esi := MockIndex{
		name:   indexName,
		types:  make(map[string]*MockIndexType),
		exists: false,
		open:   false,
	}
	return &esi
}

func (esi *MockIndex) GetVersion() string {
	return "2.2.0"
}

func (esi *MockIndex) IndexName() string {
	return esi.name
}

func (esi *MockIndex) IndexExists() (bool, error) {
	return esi.exists, nil
}

func (esi *MockIndex) TypeExists(typ string) (bool, error) {

	ok, err := esi.IndexExists()
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}
	_, ok = esi.types[typ]
	return ok, nil
}

func (esi *MockIndex) ItemExists(typeName string, id string) (bool, error) {
	ok, err := esi.TypeExists(typeName)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}
	typ := esi.types[typeName]
	_, ok = (*typ).items[id]
	return ok, nil
}

// if index already exists, does nothing
func (esi *MockIndex) Create(settings string) error {
	if esi.exists {
		return fmt.Errorf("Index already exists")
	}

	esi.exists = true

	if settings == "" {
		esi.settings = nil
		return nil
	}

	obj := map[string]interface{}{}
	err := json.Unmarshal([]byte(settings), &obj)
	if err != nil {
		return err
	}

	esi.settings = obj

	for k, v := range obj["mappings"].(map[string]interface{}) {
		mapping, err := json.Marshal(v)
		if err != nil {
			return err
		}
		err = esi.addType(k, string(mapping))
		if err != nil {
			return err
		}
	}

	return nil
}

// if index doesn't already exist, does nothing
func (esi *MockIndex) Close() error {
	esi.open = false
	return nil
}

// if index doesn't already exist, does nothing
func (esi *MockIndex) Delete() error {
	esi.exists = false
	esi.open = false

	for tk, tv := range esi.types {
		for ik := range tv.items {
			delete(tv.items, ik)
		}
		delete(esi.types, tk)
	}

	return nil
}

func (esi *MockIndex) addType(typeName string, mapping string) error {

	if mapping == "" {
		return fmt.Errorf("addType: mapping may not be null")
	}

	obj := map[string]interface{}{}
	err := json.Unmarshal([]byte(mapping), &obj)
	if err != nil {
		return err
	}

	esi.types[typeName] = &MockIndexType{
		mapping: obj,
		items:   make(map[string]*AnyType),
	}

	return nil
}

func (esi *MockIndex) SetMapping(typeName string, mapping piazza.JsonString) error {
	return esi.addType(typeName, string(mapping))
}

func (esi *MockIndex) newId() string {
	esi.idSource++
	return strconv.Itoa(esi.idSource)
}

func (esi *MockIndex) PostData(typeName string, id string, obj interface{}) (*elastic.IndexResponse, error) {
	ok, err := esi.IndexExists()
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("Index does not exist")
	}
	ok, err = esi.TypeExists(typeName)
	if err != nil {
		return nil, err
	}

	var typ *MockIndexType
	if !ok {
		typ = &MockIndexType{}
		typ.items = make(map[string]*AnyType)
		esi.types[typeName] = typ
	} else {
		typ = esi.types[typeName]
	}

	raw, err := newAnyType(id, obj)
	if err != nil {
		return nil, err
	}

	if id == "" {
		id = esi.newId()
	}

	typ.items[id] = raw

	r := &elastic.IndexResponse{Created: true, Id: id, Index: esi.name, Type: typeName}
	return r, nil
}

//TODO
func (esi *MockIndex) PutData(typeName string, id string, obj interface{}) (*elastic.IndexResponse, error) {
	return esi.PostData(typeName, id, obj)
}

func (esi *MockIndex) GetByID(typeName string, id string) (*elastic.GetResult, error) {
	ok, err := esi.ItemExists(typeName, id)
	if err != nil {
		return nil, err
	}
	if !ok {
		return &elastic.GetResult{Found: false}, fmt.Errorf("GetById: id does not exist: %s", id)
	}

	typ := esi.types[typeName]
	item := typ.items[id]
	r := &elastic.GetResult{Id: id, Source: item.raw, Found: true}
	return r, nil
}

func (esi *MockIndex) DeleteByID(typeName string, id string) (*elastic.DeleteResponse, error) {
	ok, err := esi.TypeExists(typeName)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("GetById: type does not exist: %s", typeName)
	}
	ok, err = esi.ItemExists(typeName, id)
	if err != nil {
		return nil, err
	}
	if !ok {
		return &elastic.DeleteResponse{Found: false}, err
	}

	typ := esi.types[typeName]
	delete(typ.items, id)
	r := &elastic.DeleteResponse{Found: true, Id: id}
	return r, nil
}

func (esi *MockIndex) DeleteByIDWait(typeName string, id string) (*elastic.DeleteResponse, error) {
	return esi.DeleteByID(typeName, id)
}

type srhByID []*elastic.SearchHit

func (a srhByID) Len() int {
	return len(a)
}
func (a srhByID) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}
func (a srhByID) Less(i, j int) bool {
	return (*a[i]).Id < (*a[j]).Id
}
func srhSortMatches(matches []*elastic.SearchHit) []*elastic.SearchHit {
	sort.Sort(srhByID(matches))
	return matches
}

func (esi *MockIndex) FilterByMatchAll(typeName string, realFormat *piazza.JsonPagination) (*elastic.SearchResult, error) {
	// pagination SortBy and Order are not supported!

	format := NewQueryFormat(realFormat)

	objs := make(map[string]*json.RawMessage)

	emptyResp := &elastic.SearchResult{
		Hits: &elastic.SearchHits{
			TotalHits: 0,
			MaxScore:  nil,
			Hits:      make([]*elastic.SearchHit, 0),
		},
	}

	if typeName == "" {
		if esi.types == nil {
			return emptyResp, nil
		}
		for tk, tv := range esi.types {
			if tk == percolateTypeName {
				continue
			}
			for ik, iv := range tv.items {
				objs[ik] = iv.raw
			}
		}
	} else {
		if esi.types[typeName] == nil {
			return emptyResp, nil
		}
		for ik, iv := range esi.types[typeName].items {
			objs[ik] = iv.raw
		}
	}

	resp := &elastic.SearchResult{
		Hits: &elastic.SearchHits{
			TotalHits: int64(len(objs)),
			MaxScore:  nil,
			Hits:      make([]*elastic.SearchHit, len(objs)),
		},
	}

	i := 0
	for id, obj := range objs {
		tmp := &elastic.SearchHit{
			Id:     id,
			Source: obj,
		}
		resp.Hits.Hits[i] = tmp
		i++
	}

	// TODO; sort key not supported
	// TODO: sort order not supported

	from := format.From
	size := format.Size

	resp.Hits.Hits = srhSortMatches(resp.Hits.Hits)

	if from >= len(resp.Hits.Hits) {
		resp.Hits.Hits = make([]*elastic.SearchHit, 0)
		return resp, nil
	}

	if from+size >= len(resp.Hits.Hits) {
		resp.Hits.Hits = resp.Hits.Hits[from:]
		return resp, nil
	}

	resp.Hits.Hits = resp.Hits.Hits[from : from+size]

	return resp, nil
}

func (esi *MockIndex) GetAllElements(typ string) (*elastic.SearchResult, error) {
	return nil, errors.New("GetAllElements not supported under mocking")
}

func (esi *MockIndex) FilterByMatchQuery(typ string, name string, value interface{}, realFormat *piazza.JsonPagination) (*elastic.SearchResult, error) {

	return nil, errors.New("FilterByMatchQuery not supported under mocking")
}

func (esi *MockIndex) FilterByTermQuery(typeName string, name string, value interface{}, realFormat *piazza.JsonPagination) (*elastic.SearchResult, error) {

	objs := make(map[string]*json.RawMessage)

	for ik, iv := range esi.types[typeName].items {
		objs[ik] = iv.raw
	}

	resp := &elastic.SearchResult{
		Hits: &elastic.SearchHits{
			TotalHits: 0,
			MaxScore:  nil,
			Hits:      make([]*elastic.SearchHit, 0),
		}}

	i := 0
	for id, obj := range objs {
		var iface interface{}
		err := json.Unmarshal(*obj, &iface)
		if err != nil {
			return nil, err
		}
		actualValue := iface.(map[string]interface{})[name].(string)
		if actualValue != value.(string) {
			continue
		}
		tmp := &elastic.SearchHit{
			Id:     id,
			Source: obj,
		}
		resp.Hits.Hits = append(resp.Hits.Hits, tmp)
		i++
	}

	resp.Hits.Hits = srhSortMatches(resp.Hits.Hits)
	resp.Hits.TotalHits = int64(i)

	return resp, nil
}

func (esi *MockIndex) SearchByJSON(typ string, ijsn map[string]interface{}) (*elastic.SearchResult, error) {
	dat, err := json.Marshal(ijsn)
	if err != nil {
		return nil, err
	}
	jsn := map[string]interface{}{}
	if err = piazza.UnmarshalNumber(bytes.NewReader(dat), &jsn); err != nil {
		return nil, err
	}
	var ret []*AnyType
	var aggs elastic.Aggregations = nil
	query, hasQuery := jsn["query"]
	if hasQuery {
		if ret, err = esi.query(query, esi.types[typ].items); err != nil {
			return nil, err
		}
	} else {
		ret = esi.convertToSlice(esi.types[typ].items)
	}
	iaggs, hasAggs := jsn["aggs"]
	if hasAggs {
		if aggs, err = esi.handle_aggs(iaggs, ret); err != nil {
			return nil, err
		}
	}

	size := 10
	if isize, ok := jsn["size"]; ok {
		tmp, _ := isize.(json.Number).Int64()
		size = int(tmp)
	}
	if len(ret) < size {
		size = len(ret)
	}
	ret = ret[:size]
	return esi.convertToResult(ret, aggs), nil
}

func (esi *MockIndex) GetTypes() ([]string, error) {
	var s []string

	for k := range esi.types {
		s = append(s, k)
	}

	return s, nil
}

func (esi *MockIndex) GetMapping(typ string) (interface{}, error) {
	return nil, errors.New("GetMapping not supported under mocking")
}

func (esi *MockIndex) DirectAccess(verb string, endpoint string, input interface{}, output interface{}) error {
	return fmt.Errorf("DirectAccess not supported")
}

func (esi *MockIndex) convertToSlice(any map[string]*AnyType) []*AnyType {
	res := make([]*AnyType, 0, len(any))
	for _, v := range any {
		res = append(res, v)
	}
	return res
}
func (esi *MockIndex) convertToResult(any []*AnyType, aggs elastic.Aggregations) *elastic.SearchResult {
	hits := make([]*elastic.SearchHit, 0, len(any))
	for _, item := range any {
		hits = append(hits, &elastic.SearchHit{Id: item.Id, Source: item.raw})
	}
	return &elastic.SearchResult{Hits: &elastic.SearchHits{int64(len(hits)), nil, hits}, Aggregations: aggs}
}

func (esi *MockIndex) handle_aggs(aggs interface{}, items []*AnyType) (elastic.Aggregations, error) {
	if len(aggs.(map[string]interface{})) != 1 {
		return nil, fmt.Errorf("Malformed agg query")
	}
	var aggName string
	var agg map[string]interface{}
	for k, v := range aggs.(map[string]interface{}) {
		aggName = k
		agg = v.(map[string]interface{})
		break
	}
	if len(agg) != 1 {
		return nil, fmt.Errorf("Cannot handle this agg query")
	}
	terms, ok := agg["terms"]
	if !ok {
		return nil, fmt.Errorf("Cannot handle this agg query")
	}
	ifield, ok := terms.(map[string]interface{})["field"]
	if !ok {
		return nil, fmt.Errorf("Agg terms needs a field")
	}
	field := ifield.(string)

	keys := map[interface{}]int64{}
	for _, i := range items {
		if value, ok := i.vars[field]; ok {
			if piazza.ValueIsValidArray(value) {
				vals := value.([]interface{})
				for _, v := range vals {
					if _, ok = keys[v]; !ok {
						keys[v] = 1
					} else {
						keys[v]++
					}
				}
			} else {
				if _, ok = keys[value]; !ok {
					keys[value] = 1
				} else {
					keys[value]++
				}
			}
		}
	}
	buckets := make([]*elastic.AggregationBucketKeyItem, 0, len(keys))
	for k, i := range keys {
		buckets = append(buckets, &elastic.AggregationBucketKeyItem{Key: k, DocCount: i})
	}
	dat, err := json.Marshal(elastic.AggregationBucketKeyItems{Buckets: buckets})
	if err != nil {
		return nil, err
	}
	jsn := json.RawMessage(dat)
	res := elastic.Aggregations{
		aggName: &jsn,
	}
	return res, nil
}

func (esi *MockIndex) query(query interface{}, items map[string]*AnyType) ([]*AnyType, error) {
	var ret []*AnyType
	term, hasTerm := query.(map[string]interface{})["term"]
	boool, hasBool := query.(map[string]interface{})["bool"]
	terms, hasTerms := query.(map[string]interface{})["terms"]
	if len(query.(map[string]interface{})) == 0 {
		ret = esi.convertToSlice(items)
	} else if hasTerm && hasBool || hasTerms && hasBool {
		return nil, fmt.Errorf("Cannot use both term(s) and bool in mock SearchByJSON")
	} else if hasTerm {
		ret = esi.convertToSlice(esi.term_query(term, items))
	} else if hasTerms {
		ret = esi.convertToSlice(esi.terms_query(terms, items))
	} else if hasBool {
		ret = esi.convertToSlice(esi.bool_query(boool, items))
	} else {
		return nil, fmt.Errorf("Unsupported operation in mock SearchByJSON")
	}
	return ret, nil
}

func (esi *MockIndex) term_query(term interface{}, items map[string]*AnyType) map[string]*AnyType {
	hits := map[string]*AnyType{}
	var key string
	var value interface{}
	for key, value = range term.(map[string]interface{}) {
		break
	}
	for id, item := range items {
		if v, ok := item.vars[key]; ok {
			if v == value {
				hits[id] = item
			}
		}
	}
	return hits
}
func (esi *MockIndex) terms_query(terms interface{}, items map[string]*AnyType) map[string]*AnyType {
	hits := map[string]*AnyType{}
	var key string
	var ivalues interface{}
	var values []interface{}
	for key, ivalues = range terms.(map[string]interface{}) {
		break
	}
	values = ivalues.([]interface{})
	for id, item := range items {
		if v, ok := item.vars[key]; ok {
			for _, vv := range values {
				if v == vv {
					hits[id] = item
				}
			}
		}
	}
	return hits
}
func (esi *MockIndex) bool_query(boool interface{}, items map[string]*AnyType) map[string]*AnyType {
	must, hasMust := boool.(map[string]interface{})["must"]
	should, hasShould := boool.(map[string]interface{})["should"]
	mustNot, hasMustNot := boool.(map[string]interface{})["must_not"]
	var anyTypes map[string]*AnyType
	if hasMust {
		anyTypes = esi.bool_must_query(must, items)
	} else {
		anyTypes = items
	}
	if hasShould {
		anyTypes = esi.bool_should_query(should, anyTypes)
	}
	if hasMustNot {
		anyTypes = esi.bool_must_not_query(mustNot, anyTypes)
	}
	return anyTypes
}
func (esi *MockIndex) bool_must_query(must interface{}, items map[string]*AnyType) map[string]*AnyType {
	finds := make([][]string, len(must.([]interface{})), len(must.([]interface{})))
	add := func(i int, found map[string]*AnyType) {
		finds[i] = make([]string, 0, len(found))
		for k, _ := range found {
			finds[i] = append(finds[i], k)
		}
	}
	for i, keys := range must.([]interface{}) {
		if term, hasTerm := keys.(map[string]interface{})["term"]; hasTerm {
			add(i, esi.term_query(term, items))
		} else if terms, hasTerms := keys.(map[string]interface{})["terms"]; hasTerms {
			add(i, esi.terms_query(terms, items))
		} else {
			finds[i] = []string{}
		}
	}
	contains := func(slice []string, str string) bool {
		for _, v := range slice {
			if v == str {
				return true
			}
		}
		return false
	}
	intersection := func(a, b []string) []string {
		res := []string{}
		for _, v := range a {
			if contains(b, v) {
				res = append(res, v)
			}
		}
		return res
	}

	res := map[string]*AnyType{}
	if len(finds) == 0 {
		return res
	}
	found := finds[0]
	for i := 1; i < len(finds); i++ {
		found = intersection(found, finds[i])
	}
	for _, v := range found {
		res[v] = items[v]
	}
	return res
}
func (esi *MockIndex) bool_should_query(should interface{}, items map[string]*AnyType) map[string]*AnyType {
	finds := make([][]string, len(should.([]interface{})), len(should.([]interface{})))
	add := func(i int, found map[string]*AnyType) {
		finds[i] = make([]string, 0, len(found))
		for k, _ := range found {
			finds[i] = append(finds[i], k)
		}
	}
	for i, keys := range should.([]interface{}) {
		if term, hasTerm := keys.(map[string]interface{})["term"]; hasTerm {
			add(i, esi.term_query(term, items))
		} else if terms, hasTerms := keys.(map[string]interface{})["terms"]; hasTerms {
			add(i, esi.terms_query(terms, items))
		} else {
			finds[i] = []string{}
		}
	}
	allKeys := map[string]struct{}{}
	for _, find := range finds {
		for _, k := range find {
			allKeys[k] = struct{}{}
		}
	}
	res := map[string]*AnyType{}
	for k, _ := range allKeys {
		res[k] = items[k]
	}
	return res
}
func (esi *MockIndex) bool_must_not_query(must_not interface{}, items map[string]*AnyType) map[string]*AnyType {
	res := map[string]*AnyType{}
	for k, v := range items {
		res[k] = v
	}
	musts := esi.bool_must_query(must_not, items)
	for k, _ := range musts {
		delete(res, k)
	}
	return res
}
