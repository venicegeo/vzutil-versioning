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

	"github.com/venicegeo/pz-gocommon/elasticsearch/elastic-5-api"
)

type SearchResultHit struct {
	ID     string
	Source *json.RawMessage
}

type SearchResult struct {
	totalHits int64 // total number of hits overall
	hits      []*SearchResultHit
	Found     bool
}

func NewSearchResult(searchResult *elastic.SearchResult) *SearchResult {
	numHits := len(searchResult.Hits.Hits)
	totalHits := searchResult.Hits.TotalHits

	resp := &SearchResult{
		totalHits: totalHits,
		hits:      make([]*SearchResultHit, numHits),
		Found:     true,
	}

	for i, hit := range searchResult.Hits.Hits {
		tmp := &SearchResultHit{
			ID:     hit.Id,
			Source: hit.Source,
		}
		resp.hits[i] = tmp
	}

	return resp
}

func (r *SearchResult) TotalHits() int64 {
	return r.totalHits
}

func (r *SearchResult) NumHits() int {
	return len(r.hits)
}

func (r *SearchResult) GetHits() *[]*SearchResultHit {
	return &r.hits
}

func (r *SearchResult) GetHit(i int) *SearchResultHit {
	arr := r.GetHits()
	return (*arr)[i]
}

type IndexResponse struct {
	Created bool
	ID      string
	Index   string
	Type    string
	Version int
}

func NewIndexResponse(indexResponse *elastic.IndexResponse) *IndexResponse {
	resp := &IndexResponse{
		Created: indexResponse.Created,
		ID:      indexResponse.Id,
		Index:   indexResponse.Index,
		Type:    indexResponse.Type,
		Version: indexResponse.Version,
	}
	return resp
}

// DeleteResponse is the response when a deletion of a document or type occurs
type DeleteResponse struct {
	Found bool
	ID    string
}

// NewDeleteResponse is the initializing constructor for DeleteResponse
func NewDeleteResponse(deleteResponse *elastic.DeleteResponse) *DeleteResponse {
	resp := &DeleteResponse{
		Found: deleteResponse.Found,
		ID:    deleteResponse.Id,
	}
	return resp
}

type GetResult struct {
	ID     string
	Source *json.RawMessage
	Found  bool
}

func NewGetResult(getResult *elastic.GetResult) *GetResult {
	resp := &GetResult{
		ID:     getResult.Id,
		Source: getResult.Source,
		Found:  getResult.Found,
	}
	return resp
}
