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

// This test only works if you have a local Elasticsearch installed somewhere.

package elasticsearch

import (
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"
	"testing"
	"time"

	"gopkg.in/olivere/elastic.v3"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/venicegeo/pz-gocommon/elasticsearch"
	"github.com/venicegeo/pz-gocommon/gocommon"
)

type EsTester struct {
	suite.Suite
	url string
}

func (suite *EsTester) SetupSuite() {
	suite.url = "http://localhost:9200"
}

func (suite *EsTester) TearDownSuite() {}

func TestRunSuite(t *testing.T) {
	s1 := new(EsTester)
	suite.Run(t, s1)
}

type Obj struct {
	ID   string `json:"id" binding:"required"`
	Data string `json:"data" binding:"required"`
	Tags string `json:"tags" binding:"required"`
}

const objType = "Obj"
const objMapping = `{
		"Obj":{
			"properties":{
				"id": {
					"type":"string"
				},
				"data": {
					"type":"string"
				},
				"tags": {
					"type":"string"
				}
			}
		}
	}`

var objs = []Obj{
	{ID: "id0", Data: "data0", Tags: "foo bar"},
	{ID: "id1", Data: "data1", Tags: "bar baz"},
	{ID: "id2", Data: "data2", Tags: "foo"},
}

func uniq() string {
	return fmt.Sprintf("%d", time.Now().Nanosecond())
}

func closer(t *testing.T, esi elasticsearch.IIndex) {
	if esi == nil {
		return
	}
	err := esi.Close()
	assert.NoError(t, err)
	err = esi.Delete()
	assert.NoError(t, err)
}

func (suite *EsTester) SetUpIndex() elasticsearch.IIndex {
	t := suite.T()
	assert := assert.New(t)

	var esi *elasticsearch.Index
	var err error
	var ok bool

	{
		esi, err = elasticsearch.NewIndex2(suite.url, "estest$"+uniq(), "")
		assert.NoError(err)

		_ = esi.Delete()
		//assert.NoError(err)

		ok, err = esi.IndexExists()
		assert.NoError(err)
		assert.False(ok)

		// make the index
		err = esi.Create("")
		assert.NoError(err)
		ok, err = esi.IndexExists()
		assert.NoError(err)
		assert.True(ok)

		err = esi.SetMapping(objType, objMapping)
		assert.NoError(err)
	}

	// populate the index
	for _, o := range objs {
		indexResult, err := esi.PostData(objType, o.ID, o)
		assert.NoError(err)
		assert.NotNil(indexResult)
	}

	// allow the database time to settle
	pollingFn := elasticsearch.GetData(func() (bool, error) {
		getResult, err := esi.FilterByMatchAll(objType, nil)
		if err != nil {
			return false, err
		}
		if getResult != nil && len(*getResult.GetHits()) == len(objs) {
			return true, nil
		}
		return false, nil
	})

	{
		_, err := elasticsearch.PollFunction(pollingFn)
		assert.NoError(err)
	}

	return esi
}

//---------------------------------------------------------------------------

func hasKnownPrefix(v string) bool {
	return strings.HasPrefix(v, "alerts.") ||
		strings.HasPrefix(v, "triggers.") ||
		strings.HasPrefix(v, "events.") ||
		strings.HasPrefix(v, "eventtypes.") ||
		strings.HasPrefix(v, "estest.") ||
		strings.HasPrefix(v, "test.") ||
		strings.HasPrefix(v, "getall.") ||
		strings.HasPrefix(v, "pzlogger.")
}

func deleteOldIndexes(es *elastic.Client) {
	s, err := es.IndexNames()
	if err != nil {
		panic(err)
	}
	log.Printf("%d indexes", len(s))

	del := func(nam string) {
		ret, err := es.DeleteIndex(nam).Do()
		if err != nil {
			log.Printf("%s: %s", nam, err.Error())
		} else {
			log.Printf("%s: %t", nam, ret.Acknowledged)
		}
	}

	for _, v := range s {
		if hasKnownPrefix(v) {
			del(v)
		} else {
			log.Printf("Skipping %s", v)
		}
	}

	panic(999)
}

func (suite *EsTester) Test01Client() {
	t := suite.T()
	assert := assert.New(t)

	esi, err := elasticsearch.NewIndex2(suite.url, "estest01$", "")
	assert.NoError(err)

	version := esi.GetVersion()
	assert.NoError(err)
	assert.True(strings.HasPrefix(version, "2.2") ||
		strings.HasPrefix(version, "2.3") ||
		strings.HasPrefix(version, "2.4"))
}

func (suite *EsTester) Test02SimplePost() {
	t := suite.T()
	assert := assert.New(t)

	var err error

	esi := suite.SetUpIndex()
	assert.NotNil(esi)

	defer closer(t, esi)

	type Obj2 struct {
		ID2   int    `json:"id" binding:"required"`
		Data2 string `json:"data" binding:"required"`
		Foo2  bool   `json:"foo" binding:"required"`
	}

	const mapping = `{
			"Obj2":{
				"properties":{
					"id2": {
						"type":"integer"
					},
					"data2": {
						"type":"string"
					},
					"foo2": {
						"type":"boolean"
					}
				}
			}
		}`

	typ := "Obj2"
	err = esi.SetMapping(typ, piazza.JsonString(mapping))
	assert.NoError(err)
	{
		// post it
		obj := Obj2{ID2: 99, Data2: "quick fox", Foo2: true}

		indexResult, err := esi.PostData(typ, "99", obj)
		assert.NoError(err)
		assert.NotNil(indexResult)
	}

	{
		// get it
		getResult, err := esi.GetByID(typ, "99")
		assert.NoError(err)
		assert.NotNil(getResult)
		src := getResult.Source
		assert.NotNil(src)
		var tmp Obj2
		err = json.Unmarshal(*src, &tmp)
		assert.NoError(err)
		assert.EqualValues("quick fox", tmp.Data2)
	}
}

func (suite *EsTester) Test03Operations() {
	t := suite.T()
	assert := assert.New(t)

	var tmp1, tmp2 Obj
	var src *json.RawMessage

	esi := suite.SetUpIndex()
	assert.NotNil(esi)
	defer closer(t, esi)

	{
		// GET a specific one
		getResult, err := esi.GetByID(objType, "id1")
		assert.NoError(err)
		assert.NotNil(getResult)
		src = getResult.Source
		assert.NotNil(src)
		err = json.Unmarshal(*src, &tmp1)
		assert.NoError(err)
		assert.EqualValues("data1", tmp1.Data)
	}

	{
		// SEARCH for everything
		// TODO: exercise sortKey
		searchResult, err := esi.FilterByMatchAll(objType, nil)
		assert.NoError(err)
		assert.NotNil(searchResult)

		assert.Equal(int64(3), searchResult.TotalHits())

		m := make(map[string]Obj)

		for _, hit := range *searchResult.GetHits() {
			err = json.Unmarshal(*hit.Source, &tmp1)
			assert.NoError(err)
			m[tmp1.ID] = tmp1
		}

		assert.Contains(m, "id0")
		assert.Contains(m, "id1")
		assert.Contains(m, "id2")
	}

	{
		// SEARCH for a specific one
		searchResult, err := esi.FilterByTermQuery(objType, "id", "id1", nil)
		assert.NoError(err)
		assert.NotNil(searchResult)
		assert.EqualValues(1, searchResult.TotalHits())
		hit := *searchResult.GetHit(0)
		assert.NotNil(hit)
		src = hit.Source
		assert.NotNil(src)
		err = json.Unmarshal(*src, &tmp1)
		assert.NoError(err)
		assert.EqualValues("data1", tmp1.Data)
	}

	{
		// SEARCH fuzzily
		searchResult, err := esi.FilterByTermQuery(objType, "tags", "foo", nil)
		assert.NoError(err)
		assert.NotNil(searchResult)
		assert.EqualValues(2, searchResult.TotalHits())

		hit0 := *searchResult.GetHit(0)
		assert.NotNil(hit0)
		src = hit0.Source
		assert.NotNil(src)
		err = json.Unmarshal(*src, &tmp1)
		assert.NoError(err)

		hit1 := *searchResult.GetHit(1)
		src = hit1.Source
		assert.NotNil(src)
		err = json.Unmarshal(*src, &tmp2)
		assert.NoError(err)

		ok1 := ("id0" == tmp1.ID && "id2" == tmp2.ID)
		ok2 := ("id0" == tmp2.ID && "id2" == tmp1.ID)
		assert.True((ok1 || ok2) && !(ok1 && ok2))
	}

	{
		// DELETE by id
		_, err := esi.DeleteByID(objType, "id2")
		assert.NoError(err)
		_, err = esi.GetByID(objType, "id2")
		assert.Error(err)
	}
}

func (suite *EsTester) Test04JsonOperations() {
	t := suite.T()
	assert := assert.New(t)

	var tmp1, tmp2 Obj
	var err error
	var src *json.RawMessage

	var searchResult *elasticsearch.SearchResult

	esi := suite.SetUpIndex()
	assert.NotNil(esi)
	defer closer(t, esi)

	// SEARCH for everything
	{
		str :=
			`{
				"query": {
					"match_all": {}
				}
			}`

		searchResult, err = esi.SearchByJSON(objType, str)
		assert.NoError(err)
		assert.NotNil(searchResult)

		for _, hit := range *searchResult.GetHits() {
			err = json.Unmarshal(*hit.Source, &tmp1)
			assert.NoError(err)
		}
		assert.EqualValues(3, searchResult.TotalHits())
	}

	// SEARCH for a specific one
	{
		str :=
			`{
				"query": {
					"term": {"id":"id1"}
				}
			}`

		searchResult, err = esi.SearchByJSON(objType, str)
		assert.NoError(err)
		assert.NotNil(searchResult)

		assert.EqualValues(1, searchResult.TotalHits())
		src = searchResult.GetHit(0).Source
		assert.NotNil(src)
		err = json.Unmarshal(*src, &tmp1)
		assert.NoError(err)
		assert.EqualValues("data1", tmp1.Data)
	}

	// SEARCH fuzzily
	{
		str :=
			`{
				"query": {
					"term": {"tags":"foo"}
				}
			}`

		searchResult, err = esi.SearchByJSON(objType, str)
		assert.NoError(err)
		assert.NotNil(searchResult)

		assert.EqualValues(2, searchResult.TotalHits())
		hit0 := searchResult.GetHit(0)
		assert.NotNil(hit0)

		src = hit0.Source
		assert.NotNil(src)
		err = json.Unmarshal(*src, &tmp1)
		assert.NoError(err)

		hit1 := searchResult.GetHit(1)
		assert.NotNil(hit1)
		src = hit1.Source
		assert.NotNil(src)
		err = json.Unmarshal(*src, &tmp2)
		assert.NoError(err)

		ok1 := ("id0" == tmp1.ID && "id2" == tmp2.ID)
		ok2 := ("id0" == tmp2.ID && "id2" == tmp1.ID)
		assert.True((ok1 || ok2) && !(ok1 && ok2))
	}
}

func (suite *EsTester) Test05Mapping() {
	t := suite.T()
	assert := assert.New(t)

	var err error

	esi := suite.SetUpIndex()
	assert.NotNil(esi)
	defer closer(t, esi)

	mapping :=
		`{
			"tweetdoc":{
				"properties":{
					"message":{
						"type":"string",
						"store":true
					}
				}
			}
		}`

	err = esi.SetMapping("tweetdoc", piazza.JsonString(mapping))
	assert.NoError(err)

	mappings, err := esi.GetMapping("tweetdoc")
	assert.NoError(err)

	tweetdoc := mappings.(map[string]interface{})["tweetdoc"]
	assert.NotNil(tweetdoc)
	properties := tweetdoc.(map[string]interface{})["properties"]
	assert.NotNil(properties)
	message := properties.(map[string]interface{})["message"]
	assert.NotNil(message)
	typ := message.(map[string]interface{})["type"].(string)
	assert.NotNil(typ)
	store := message.(map[string]interface{})["store"].(bool)
	assert.NotNil(store)

	assert.EqualValues("string", typ)
	assert.True(store)
}

func (suite *EsTester) Test06SetMapping() {
	t := suite.T()
	assert := assert.New(t)

	esi := suite.SetUpIndex()
	assert.NotNil(esi)
	defer closer(t, esi)

	var err error

	var expected piazza.JsonString
	expected = `{
			"MyTestObj": {
				"properties":{
					"bool1": {"type": "boolean"},
					"date1": {"format": "dateOptionalTime", "type": "date"},
					"double1": {"type": "double"},
					"integer1": {"type": "integer"},
					"integer2": {"type": "integer"}
				}
			}
		}`

	err = esi.SetMapping("MyTestObj", expected)
	assert.NoError(err)

	mapobj, err := esi.GetMapping("MyTestObj")
	assert.NoError(err)

	actual, err := json.Marshal(mapobj)
	assert.NoError(err)
	assert.JSONEq(string(expected), string(actual))

	mappings, err := esi.GetTypes()
	assert.NoError(err)
	assert.Len(mappings, 2)
	assert.True((mappings[0] == "Obj" && mappings[1] == "MyTestObj") ||
		(mappings[1] == "Obj" && mappings[0] == "MyTestObj"))
}

func (suite *EsTester) Test07ConstructMapping() {
	t := suite.T()
	assert := assert.New(t)

	esi := suite.SetUpIndex()
	assert.NotNil(esi)
	defer closer(t, esi)

	items := make(map[string]elasticsearch.MappingElementTypeName)

	items["integer1"] = elasticsearch.MappingElementTypeInteger
	items["integer2"] = elasticsearch.MappingElementTypeInteger
	items["double1"] = elasticsearch.MappingElementTypeDouble
	items["bool1"] = elasticsearch.MappingElementTypeBool
	items["date1"] = elasticsearch.MappingElementTypeDate

	jsonstr, err := elasticsearch.ConstructMappingSchema("MyTestObj", items)
	assert.NoError(err)
	assert.NotNil(jsonstr)
	assert.NotEmpty(jsonstr)

	var iface interface{}
	err = json.Unmarshal([]byte(jsonstr), &iface)
	assert.NoError(err)

	byts, err := json.Marshal(iface)
	assert.NoError(err)
	assert.NotNil(byts)

	actual := string(byts)

	expected :=
		`{"MyTestObj":{"properties":{"bool1":{"type":"boolean"},"date1":{"type":"date"},"double1":{"type":"double"},"integer1":{"type":"integer"},"integer2":{"type":"integer"}}}}`

	assert.Equal(expected, actual)

	err = esi.SetMapping("MyTestObj", piazza.JsonString(actual))
	assert.NoError(err)
}

func (suite *EsTester) Test08Percolation() {
	t := suite.T()
	assert := assert.New(t)

	esi := suite.SetUpIndex()
	assert.NotNil(esi)
	defer closer(t, esi)

	items := make(map[string]elasticsearch.MappingElementTypeName)
	items["tag"] = elasticsearch.MappingElementTypeString
	jsonstr, err := elasticsearch.ConstructMappingSchema("Event", items)
	assert.NoError(err)
	assert.NotEmpty(jsonstr)

	err = esi.SetMapping("Event", jsonstr)
	assert.NoError(err)

	query1 :=
		`{
		"query": {
			"match": {
				"tag": {
					"query": "kitten"
				}
			}
		}
	}`
	query2 :=
		`{
		"query" : {
			"match" : {
				"tag" : "lemur"
			}
		}
	}`

	_, err = esi.AddPercolationQuery("p1", piazza.JsonString(query1))
	assert.NoError(err)

	type Event struct {
		ID  string `json:"id" binding:"required"`
		Tag string `json:"tag" binding:"required"`
	}
	event1 := Event{ID: "id1", Tag: "kitten"}
	event2 := Event{ID: "id2", Tag: "cat"}
	event3 := Event{ID: "id3", Tag: "lemur"}

	percolateResponse, err := esi.AddPercolationDocument("Event", event1)
	assert.NoError(err)
	assert.EqualValues(1, percolateResponse.Total)
	assert.Equal("p1", percolateResponse.Matches[0].Id)

	percolateResponse, err = esi.AddPercolationDocument("Event", event2)
	assert.NoError(err)
	assert.EqualValues(0, percolateResponse.Total)

	_, err = esi.AddPercolationQuery("p2", piazza.JsonString(query2))
	assert.NoError(err)

	percolateResponse, err = esi.AddPercolationDocument("Event", event3)
	assert.NoError(err)
	assert.EqualValues(1, percolateResponse.Total)
	assert.Equal("p2", percolateResponse.Matches[0].Id)
}

type ByID []*elasticsearch.PercolateResponseMatch

func (a ByID) Len() int {
	return len(a)
}
func (a ByID) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}
func (a ByID) Less(i, j int) bool {
	return a[i].Id < a[j].Id
}
func sortMatches(matches []*elasticsearch.PercolateResponseMatch) []*elasticsearch.PercolateResponseMatch {
	sort.Sort(ByID(matches))
	return matches
}

func (suite *EsTester) Test09FullPercolation() {
	t := suite.T()
	assert := assert.New(t)

	var esi elasticsearch.IIndex

	// create index
	{
		var err error

		esi, err = elasticsearch.NewIndex2(suite.url, "estest09$"+uniq(), "")
		assert.NoError(err)

		defer closer(t, esi)

		// make the index
		err = esi.Create("")
		assert.NoError(err)

		ok, err := esi.IndexExists()
		assert.NoError(err)
		assert.True(ok)
	}

	//-----------------------------------------------------------------------

	addMappings := func(maps map[string](map[string]elasticsearch.MappingElementTypeName)) {
		for k, v := range maps {
			jsn, err := elasticsearch.ConstructMappingSchema(k, v)
			assert.NoError(err)
			assert.NotEmpty(jsn)
			err = esi.SetMapping(k, jsn)
			assert.NoError(err)
		}
	}

	addQueries := func(queries map[string]piazza.JsonString) {
		for k, v := range queries {
			_, err := esi.AddPercolationQuery(k, v)
			assert.NoError(err)
		}
	}

	type TestCase struct {
		typeName string
		event    interface{}
		expected []string
	}

	addEvents := func(tests []TestCase) {
		for i, t := range tests {

			percolateResponse, err := esi.AddPercolationDocument(t.typeName, t.event)
			assert.NoError(err)

			assert.EqualValues(len(t.expected), percolateResponse.Total, fmt.Sprintf("for test #%d", i))

			matches := sortMatches(percolateResponse.Matches)

			for i, expected := range t.expected {
				assert.Equal(esi.IndexName(), matches[i].Index)
				assert.Equal(expected, matches[i].Id)
			}
		}
	}

	//-----------------------------------------------------------------------

	type EventType1 struct {
		ID  string `json:"id" binding:"required"`
		Str string `json:"str" binding:"required"`
		Num int    `json:"num" binding:"required"`
	}

	type EventType2 struct {
		ID  string `json:"id" binding:"required"`
		Boo bool   `json:"boo" binding:"required"`
		Num int    `json:"num" binding:"required"`
	}

	maps := map[string](map[string]elasticsearch.MappingElementTypeName){
		"EventType1": map[string]elasticsearch.MappingElementTypeName{
			"id":  elasticsearch.MappingElementTypeString,
			"str": elasticsearch.MappingElementTypeString,
			"num": elasticsearch.MappingElementTypeInteger,
		},
		"EventType2": map[string]elasticsearch.MappingElementTypeName{
			"id":  elasticsearch.MappingElementTypeString,
			"boo": elasticsearch.MappingElementTypeBool,
			"num": elasticsearch.MappingElementTypeInteger,
		},
	}
	addMappings(maps)

	//-----------------------------------------------------------------------

	q1 :=
		`{
			"query": {
				"match": {
					"str": {
						"query": "kitten"
					}
				}
			}
		}`
	q2 :=
		`{
			"query" : {
				"match" : {
					"boo" : true
				}
			}
		}`
	q3 :=
		`{
			"query" : {
				"match" : {
					"num" : 17
				}
			}
		}`
	q4 :=
		`{
			"query" : {
				"range" : {
					"num" : {
						"lt": 10.0
					}
				}
			}
		}`
	q5 :=
		`{
			"query" : {
				"filtered": {
					"query": {
						"match": {
							"num": 17
						}
					},
					"filter": {
						"term": {
							"_type": "EventType2"
						}
					}
				}
			}
		}`
	q6 :=
		`{
		"query" : {
			"bool": {
				"must": [
					{
						"match" : {
							"num" : 17
						}
					},
					{
						"match" : {
							"_type" : "EventType1"
						}
					}
				]
			}
		}
	}`
	queries := map[string]piazza.JsonString{
		"Q1": piazza.JsonString(q1),
		"Q2": piazza.JsonString(q2),
		"Q3": piazza.JsonString(q3),
		"Q4": piazza.JsonString(q4),
		"Q5": piazza.JsonString(q5),
		"Q6": piazza.JsonString(q6),
	}
	addQueries(queries)

	//-----------------------------------------------------------------------

	tests := []TestCase{
		TestCase{
			typeName: "EventType1",
			event:    EventType1{ID: "E1", Str: "kitten", Num: 17},
			expected: []string{"Q1", "Q3", "Q6"},
		},
		TestCase{
			typeName: "EventType2",
			event:    EventType2{ID: "E2", Boo: true, Num: 17},
			expected: []string{"Q2", "Q3", "Q5"},
		},
		TestCase{
			typeName: "EventType1",
			event:    EventType1{ID: "E3", Str: "lemur", Num: -31},
			expected: []string{"Q4"},
		},
	}

	addEvents(tests)
}

func (suite *EsTester) Test10GetAll() {
	t := suite.T()
	assert := assert.New(t)

	esi, err := elasticsearch.NewIndex2(suite.url, "getall$", "")
	assert.NoError(err)
	defer closer(t, esi)

	// make the index
	err = esi.Create("")
	assert.NoError(err)

	type T1 struct {
		Data1  string `json:"data1" binding:"required"`
		Extra1 string `json:"extra1" binding:"required"`
	}

	type T2 struct {
		Data2  int    `json:"data2" binding:"required"`
		Extra2 string `json:"extra2" binding:"required"`
	}

	schema1 :=
		`{
			"schema1":{
				"properties":{
					"data1":{
						"type":"string",
						"store":true
					},
					"extra1":{
						"type":"string",
						"store":true
					}
				}
			}
		}`

	schema2 :=
		`{
			"schema2":{
				"properties":{
					"data2":{
						"type":"integer",
						"store":true
					},
					"extra2":{
						"type":"string",
						"store":true
					}
				}
			}
		}`

	err = esi.SetMapping("schema1", piazza.JsonString(schema1))
	assert.NoError(err)
	err = esi.SetMapping("schema2", piazza.JsonString(schema2))
	assert.NoError(err)

	obj1 := T1{Data1: "obj", Extra1: "extra1"}
	obj2 := T2{Data2: 123, Extra2: "extra2"}
	indexResult, err := esi.PostData("schema1", "id1", obj1)
	assert.NoError(err)
	assert.NotNil(indexResult)
	indexResult, err = esi.PostData("schema2", "id2", obj2)
	assert.NoError(err)
	assert.NotNil(indexResult)

	{
		// GET a specific one
		getResult, err := esi.GetByID("schema1", "id1")
		assert.NoError(err)
		assert.NotNil(getResult)
		src := getResult.Source
		assert.NotNil(src)
		var tmp T1
		err = json.Unmarshal(*src, &tmp)
		assert.NoError(err)
		assert.EqualValues("obj", tmp.Data1)
	}

	{
		// GET a specific one
		getResult, err := esi.GetByID("schema2", "id2")
		assert.NoError(err)
		assert.NotNil(getResult)
		src := getResult.Source
		assert.NotNil(src)
		var tmp T2
		err = json.Unmarshal(*src, &tmp)
		assert.NoError(err)
		assert.Equal(123, tmp.Data2)
	}

	{
		// GET the types
		strs, err := esi.GetTypes()
		assert.NoError(err)
		assert.Len(strs, 2)
		if strs[0] == "schema1" {
			assert.EqualValues("schema2", strs[1])
		} else if strs[0] == "schema2" {
			assert.EqualValues("schema1", strs[1])
		} else {
			assert.True(false)
		}
	}

	{
		pollingFn := elasticsearch.GetData(func() (bool, error) {
			getResult, err := esi.FilterByMatchAll("", nil)
			if err != nil {
				return false, err
			}
			if getResult != nil && len(*getResult.GetHits()) == 2 {
				return true, nil
			}
			return false, nil
		})

		_, err := elasticsearch.PollFunction(pollingFn)
		assert.NoError(err)
		getResult, err := esi.FilterByMatchAll("", nil)
		assert.NoError(err)
		assert.NotNil(getResult)
		assert.Len(*getResult.GetHits(), 2)
		src1 := getResult.GetHit(0).Source
		assert.NotNil(src1)
		src2 := getResult.GetHit(1).Source
		assert.NotNil(src2)

		var tmp1 T1
		var tmp2 T2
		err1 := json.Unmarshal(*src1, &tmp1)
		err2 := json.Unmarshal(*src2, &tmp2)
		assert.True((err1 == nil && err2 == nil) || (err1 != nil && err2 != nil))

		if err1 != nil {
			err = json.Unmarshal(*src1, &tmp1)
			assert.NoError(err)
			err = json.Unmarshal(*src2, &tmp2)
			assert.NoError(err)
		} else {
			err = json.Unmarshal(*src1, &tmp2)
			assert.NoError(err)
			err = json.Unmarshal(*src2, &tmp1)
			assert.NoError(err)
		}

		assert.Equal(tmp1.Data1, "obj")
		assert.Equal(tmp1.Extra1, "extra1")
		assert.Equal(tmp2.Data2, 123)
		assert.Equal(tmp2.Extra2, "extra2")
	}
}

func (suite *EsTester) Test11Pagination() {
	t := suite.T()
	assert := assert.New(t)

	var err error

	esi := suite.SetUpIndex()
	assert.NotNil(esi)
	defer closer(t, esi)

	type Obj3 struct {
		ID3   string `json:"id3" binding:"required"`
		Data3 int    `json:"data3" binding:"required"`
	}
	obj3Mapping :=
		`{
			"Obj3":{
				"properties":{
				"id3": {
					"type":"string",
					"store":true
				},
				"data3": {
					"type":"integer",
					"store": true
				}
			}
		}
	}`

	err = esi.SetMapping("Obj3", piazza.JsonString(obj3Mapping))
	assert.NoError(err)

	p := uniq()

	for i := 0; i <= 9; i++ {
		id := fmt.Sprintf("id%d_%s", i, p)
		obj := Obj3{ID3: id, Data3: i * i}
		indexResult, err := esi.PostData("Obj3", id, obj)
		assert.NoError(err)
		assert.NotNil(indexResult)
		assert.EqualValues(id, indexResult.ID)
	}

	{
		realFormat := &piazza.JsonPagination{
			PerPage: 4,
			Page:    0,
			Order:   piazza.SortOrderAscending,
			SortBy:  "id3",
		}
		pollingFn := elasticsearch.GetData(func() (bool, error) {
			getResult, err := esi.FilterByMatchAll("Obj3", realFormat)
			if err != nil {
				return false, err
			}
			if getResult != nil && getResult.TotalHits() == 10 {
				return true, nil
			}
			return false, nil
		})

		_, err := elasticsearch.PollFunction(pollingFn)
		assert.NoError(err)
		getResult, err := esi.FilterByMatchAll("Obj3", realFormat)
		assert.NoError(err)
		assert.Len(*getResult.GetHits(), 4)
		assert.Equal("id0_"+p, getResult.GetHit(0).ID)
		assert.Equal("id1_"+p, getResult.GetHit(1).ID)
		assert.Equal("id2_"+p, getResult.GetHit(2).ID)
		assert.Equal("id3_"+p, getResult.GetHit(3).ID)
	}

	{
		realFormat := &piazza.JsonPagination{
			PerPage: 3,
			Page:    1,
			Order:   piazza.SortOrderAscending,
			SortBy:  "id3",
		}
		getResult, err := esi.FilterByMatchAll("Obj3", realFormat)
		assert.NoError(err)
		assert.Len(*getResult.GetHits(), 3)
		assert.Equal("id3_"+p, getResult.GetHit(0).ID)
		assert.Equal("id4_"+p, getResult.GetHit(1).ID)
		assert.Equal("id5_"+p, getResult.GetHit(2).ID)
	}
}
