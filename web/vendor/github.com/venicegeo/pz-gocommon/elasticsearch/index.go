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
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/venicegeo/pz-gocommon/gocommon"

	"github.com/venicegeo/pz-gocommon/elasticsearch/elastic-5-api"
	"golang.org/x/net/context"
)

var _ IIndex = (*Index)(nil)

// Index is a representation of the Elasticsearch index.
type Index struct {
	lib     *elastic.Client
	version string
	index   string
	url     string
	user    string
	pass    string
}

// NewIndex is the initializing constructor for the type Index.
func NewIndex(sys *piazza.SystemConfig, index string, settings string) (*Index, error) {
	url, err := sys.GetURL(piazza.PzElasticSearch)
	if err != nil {
		return nil, err
	}

	return NewIndex2(url, "", "", index, settings)
}

func NewIndex2(url, user, pass, index, settings string) (*Index, error) {
	if strings.HasSuffix(index, "$") {
		index = fmt.Sprintf("%s.%x", index[0:len(index)-1], time.Now().Nanosecond())
	}

	esi := &Index{
		index: index,
		url:   url,
		user:  user,
		pass:  pass,
	}

	var err error

	esi.lib, err = elastic.NewClient(
		elastic.SetURL(url),
		elastic.SetBasicAuth(user, pass),
		elastic.SetSniff(false),
		elastic.SetMaxRetries(5),
		//elastic.SetErrorLog(log.New(os.Stderr, "ELASTIC ", log.LstdFlags)), // TODO
		//elastic.SetInfoLog(log.New(os.Stdout, "", log.LstdFlags)),
	)
	if err != nil {
		return nil, err
	}

	esi.version, err = esi.lib.ElasticsearchVersion(url)
	if err != nil {
		return nil, err
	}

	// This does nothing if the index is already created, but creates it if not
	err = esi.Create(settings)
	if err != nil {
		return nil, err
	}

	return esi, nil
}

func IndexExists(sys *piazza.SystemConfig, index string) (bool, error) {
	url, err := sys.GetURL(piazza.PzElasticSearch)
	if err != nil {
		return false, err
	}
	if strings.HasSuffix(index, "$") {
		index = fmt.Sprintf("%s.%x", index[0:len(index)-1], time.Now().Nanosecond())
	}

	esi := &Index{
		index: index,
		url:   url,
	}

	esi.lib, err = elastic.NewClient(
		elastic.SetURL(url),
		elastic.SetSniff(false),
		elastic.SetMaxRetries(5),
		//elastic.SetErrorLog(log.New(os.Stderr, "ELASTIC ", log.LstdFlags)), // TODO
		//elastic.SetInfoLog(log.New(os.Stdout, "", log.LstdFlags)),
	)
	if err != nil {
		return false, err
	}

	esi.version, err = esi.lib.ElasticsearchVersion(url)
	if err != nil {
		return false, err
	}

	return esi.IndexExists()
}

// GetVersion returns the Elasticsearch version.
func (esi *Index) GetVersion() string {
	return esi.version
}

// IndexName returns the name of the index.
func (esi *Index) IndexName() string {
	return esi.index
}

// IndexExists checks to see if the index exists.
func (esi *Index) IndexExists() (bool, error) {
	ok, err := esi.lib.IndexExists(esi.index).Do(context.Background())
	if err != nil {
		return false, err
	}
	return ok, nil
}

// TypeExists checks to see if the specified type exists within the index.
func (esi *Index) TypeExists(typ string) (bool, error) {
	ok, err := esi.IndexExists()
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}

	ok, err = esi.lib.TypeExists().Index(esi.index).Type(typ).Do(context.Background())
	if err != nil {
		return false, err
	}
	return ok, nil
}

// ItemExists checks to see if the specified item exists within the type and index specified.
func (esi *Index) ItemExists(typ string, id string) (bool, error) {
	ok, err := esi.TypeExists(typ)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}

	ok, err = esi.lib.Exists().Index(esi.index).Type(typ).Id(id).Do(context.Background())
	if err != nil {
		return false, err
	}
	return ok, nil
}

// Create the index; if index already exists, does nothing.
func (esi *Index) Create(settings string) error {
	ok, err := esi.IndexExists()
	if err != nil {
		return err
	}
	if ok {
		//return fmt.Errorf("Index %s already exists", esi.index)
		return nil
	}

	createIndex, err := esi.lib.CreateIndex(esi.index).Body(settings).Do(context.Background())

	if err != nil {
		return err
	}

	if !createIndex.Acknowledged {
		return fmt.Errorf("elasticsearch.Index.Create: create index not acknowledged")
	}

	return nil
}

// Close the index; if index doesn't already exist, does nothing.
func (esi *Index) Close() error {
	// TODO: the caller should enforce this instead
	ok, err := esi.IndexExists()
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("Index %s does not already exist", esi.index)
	}

	_, err = esi.lib.CloseIndex(esi.index).Do(context.Background())
	return err
}

// Delete the index; if index doesn't already exist, does nothing.
func (esi *Index) Delete() error {
	ok, err := esi.IndexExists()
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("Index %s does not exist", esi.index)
	}

	deleteIndex, err := esi.lib.DeleteIndex(esi.index).Do(context.Background())
	if err != nil {
		return err
	}

	if !deleteIndex.Acknowledged {
		return fmt.Errorf("elasticsearch.Index.Delete: delete index not acknowledged")
	}
	return nil
}

// PostData send JSON data to the index.
func (esi *Index) PostData(typ string, id string, obj interface{}) (*IndexResponse, error) {
	ok, err := esi.IndexExists()
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("Index %s does not exist", esi.index)
	}
	ok, err = esi.TypeExists(typ)
	if err != nil {
		return nil, err
	}
	if !ok {
		err = fmt.Errorf("Type %s in index %s does not exist", typ, esi.index)
		return nil, err
	}

	indexResponse, err := esi.lib.Index().
		Index(esi.index).
		Type(typ).
		Id(id).
		BodyJson(obj).
		Do(context.Background())

	if err != nil {
		return nil, err
	}
	return NewIndexResponse(indexResponse), nil
}

//TODO
func (esi *Index) PutData(typ string, id string, obj interface{}) (*IndexResponse, error) {
	return esi.PostData(typ, id, obj)
}

// GetByID returns a document by ID within the specified index and type.
func (esi *Index) GetByID(typ string, id string) (*GetResult, error) {
	// TODO: the caller should enforce this instead (here and elsewhere)
	ok, err := esi.ItemExists(typ, id)
	if err != nil {
		return nil, err
	}
	if !ok {
		return &GetResult{Found: false}, fmt.Errorf("Item %s in index %s and type %s does not exist", id, esi.index, typ)
	}

	getResult, err := esi.lib.Get().Index(esi.index).Type(typ).Id(id).Do(context.Background())
	return NewGetResult(getResult), err
}

// DeleteByID deletes a document by ID within a specified index and type.
func (esi *Index) DeleteByID(typ string, id string) (*DeleteResponse, error) {
	ok, err := esi.ItemExists(typ, id)
	if err != nil {
		return nil, err
	}
	if !ok {
		return &DeleteResponse{Found: false}, fmt.Errorf("Item %s in index %s and type %s does not exist", id, esi.index, typ)
	}

	deleteResponse, err := esi.lib.Delete().
		Index(esi.index).
		Type(typ).
		Id(id).
		Do(context.Background())
	return NewDeleteResponse(deleteResponse), err
}

// DeleteByID deletes a document by ID within a specified index and type and waits before returning.
func (esi *Index) DeleteByIDWait(typ string, id string) (*DeleteResponse, error) {
	ok, err := esi.ItemExists(typ, id)
	if err != nil {
		return nil, err
	}
	if !ok {
		return &DeleteResponse{Found: false}, fmt.Errorf("Item %s in index %s and type %s does not exist", id, esi.index, typ)
	}

	deleteResponse, err := esi.lib.Delete().
		Index(esi.index).
		Type(typ).
		Id(id).
		Refresh("wait_for").
		Do(context.Background())
	return NewDeleteResponse(deleteResponse), err
}

// FilterByMatchAll returns all documents of a specified type, in the format
// specified by the realFormat parameter.
func (esi *Index) FilterByMatchAll(typ string, realFormat *piazza.JsonPagination) (*SearchResult, error) {
	// ok := typ != "" && esi.TypeExists(typ)
	// if !ok {
	// 	return nil, fmt.Errorf("Type %s in index %s does not exist", typ, esi.index)
	// }

	q := elastic.NewMatchAllQuery()
	f := esi.lib.Search().Index(esi.index).Type(typ).Query(q)

	if realFormat != nil {
		format := NewQueryFormat(realFormat)
		f = f.From(format.From)
		f = f.Size(format.Size)
		f = f.Sort(format.Key, format.Order)
	}

	searchResult, err := f.Do(context.Background())
	if err != nil {
		// if the mapping (or the index?) doesn't exist yet, squash the error
		// (this is the case in some of the unit tests which ((try to)) assure the DB is empty)
		resp := &SearchResult{totalHits: 0, hits: make([]*SearchResultHit, 0)}
		return resp, nil
	}

	resp := NewSearchResult(searchResult)
	return resp, nil
}

// GetAllElements returns all documents of a specified type.
func (esi *Index) GetAllElements(typ string) (*SearchResult, error) {
	if typ == "" {
		return nil, fmt.Errorf("elasticsearch.Index.GetAllElements: empty type")
	}

	ok, err := esi.TypeExists(typ)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("elasticsearch.Index.GetAllElements: type %s in index %s does not exist", typ, esi.index)
	}

	q := elastic.NewMatchAllQuery()
	result, err := esi.lib.Search().
		Index(esi.index).
		Type(typ).
		Query(q).
		Do(context.Background())
	if err != nil {
		return nil, err
	}

	resp := NewSearchResult(result)
	return resp, nil
}

// FilterByTermQuery creates an Elasticsearch term query and performs the query over the specified type.
// For more information on term queries, see
// https://www.elastic.co/guide/en/elasticsearch/reference/current/query-dsl-term-query.html
func (esi *Index) FilterByTermQuery(typ string, name string, value interface{}, realFormat *piazza.JsonPagination) (*SearchResult, error) {
	if typ == "" {
		return nil, fmt.Errorf("Can't filter on type \"\"")
	}
	ok, err := esi.TypeExists(typ)
	if err != nil {
		return nil, err
	}
	if !ok {
		return &SearchResult{Found: false}, fmt.Errorf("Type %s in index %s does not exist", typ, esi.index)
	}

	// Returns a query of the form {"term":{"name":"value"}}
	// The value parameter is typically sent in as a string rather than an interface,
	// but technically value can be an interface.
	termQuery := elastic.NewTermQuery(name, value)
	f := esi.lib.Search().
		Index(esi.index).
		Type(typ).
		Query(termQuery)

	if realFormat != nil {
		format := NewQueryFormat(realFormat)
		f = f.From(format.From)
		f = f.Size(format.Size)
		f = f.Sort(format.Key, format.Order)
	}

	searchResult, err := f.Do(context.Background())

	return NewSearchResult(searchResult), err
}

// FilterByMatchQuery creates an Elasticsearch match query and performs the query over the specified type.
// For more information on match queries, see
// https://www.elastic.co/guide/en/elasticsearch/reference/current/query-dsl-match-query.html
func (esi *Index) FilterByMatchQuery(typ string, name string, value interface{}, realFormat *piazza.JsonPagination) (*SearchResult, error) {
	if typ == "" {
		return nil, fmt.Errorf("Can't filter on type \"\"")
	}
	ok, err := esi.TypeExists(typ)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("Type %s in index %s does not exist", typ, esi.index)
	}

	matchQuery := elastic.NewMatchQuery(name, value)
	f := esi.lib.Search().
		Index(esi.index).
		Type(typ).
		Query(matchQuery)

	if realFormat != nil {
		format := NewQueryFormat(realFormat)
		f = f.From(format.From)
		f = f.Size(format.Size)
		f = f.Sort(format.Key, format.Order)
	}

	searchResult, err := f.Do(context.Background())

	return NewSearchResult(searchResult), err
}

// SearchByJSON performs a search over the index via raw JSON.
func (esi *Index) SearchByJSON(typ string, jsn string) (*SearchResult, error) {
	var obj interface{}
	err := json.Unmarshal([]byte(jsn), &obj)
	if err != nil {
		return nil, err
	}

	searchResult, err := esi.lib.Search().
		Index(esi.index).
		Type(typ).
		Source(obj).
		Do(context.Background())
	if err != nil {
		return nil, err
	}

	return NewSearchResult(searchResult), nil
}

// SetMapping sets the _mapping field for a new type.
func (esi *Index) SetMapping(typename string, jsn piazza.JsonString) error {

	ok, err := esi.IndexExists()
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("Index %s does not exist", esi.index)
	}

	putresp, err := esi.lib.PutMapping().Index(esi.index).Type(typename).BodyString(string(jsn)).Do(context.Background())
	if err != nil {
		return err
	}
	if putresp == nil {
		return fmt.Errorf("expected put mapping response; got: %v", putresp)
	}
	if !putresp.Acknowledged {
		return fmt.Errorf("expected put mapping ack; got: %v", putresp.Acknowledged)
	}

	return nil
}

// GetTypes returns the list of types within the index.
func (esi *Index) GetTypes() ([]string, error) {
	ok, err := esi.IndexExists()
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("Index %s does not exist", esi.index)
	}

	getresp, err := esi.lib.IndexGet().Feature("_mappings").Index(esi.index).Do(context.Background())
	if err != nil {
		return nil, err
	}

	result := []string{}
	for _, index := range getresp {
		for typ, _ := range index.Mappings {
			result = append(result, typ)
		}
	}

	return result, nil
}

// GetMapping returns the _mapping of a type.
func (esi *Index) GetMapping(typ string) (interface{}, error) {

	ok, err := esi.TypeExists(typ)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("Type %s in index %s does not exist", typ, esi.index)
	}

	getresp, err := esi.lib.GetMapping().Index(esi.index).Type(typ).Do(context.Background())
	if err != nil {
		return nil, fmt.Errorf("expected get mapping to succeed; got: %v", err)
	}
	if getresp == nil {
		return nil, fmt.Errorf("expected get mapping response; got: %v", getresp)
	}

	for _, props := range getresp {
		return props.(map[string]interface{})["mappings"], nil
	}

	return nil, fmt.Errorf("type not found after loop; got: %v", getresp)
}

func (esi *Index) DirectAccess(verb string, endpoint string, input interface{}, output interface{}) error {
	h := &piazza.Http{
		BaseUrl:       esi.url,
		BasicAuthUser: esi.user,
		BasicAuthPass: esi.pass,
	}
	_, err := h.Verb(verb, endpoint, input, output)
	return err
}
