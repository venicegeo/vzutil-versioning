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

package piazza

import "fmt"

//----------------------------------------------------------

// SortOrder indicates ascending (1,2,3) or descending (3,2,1) order.
type SortOrder string

const (
	// SortOrderAscending is for "a, b, c, ..."
	SortOrderAscending SortOrder = "asc"

	// SortOrderDescending is for "z, y, x, ..."
	SortOrderDescending SortOrder = "desc"
)

// JsonPagination is the Piazza model for pagination json responses.
type JsonPagination struct {
	Count   int       `json:"count"` // only used when writing output
	Page    int       `json:"page"`
	PerPage int       `json:"perPage"`
	SortBy  string    `json:"sortBy"`
	Order   SortOrder `json:"order"`
}

var defaultJsonPagination = &JsonPagination{
	PerPage: 10,
	Page:    0,
	Order:   SortOrderDescending,
	SortBy:  "createdOn",
}

// NewJsonPagination creates a JsonPagination object. The default values will
// be overwritten with any appropriate values from the params list.
func NewJsonPagination(params *HttpQueryParams) (*JsonPagination, error) {

	jp := &JsonPagination{}

	perPage, err := params.GetPerPage(defaultJsonPagination.PerPage)
	if err != nil {
		return nil, err
	}
	jp.PerPage = perPage

	page, err := params.GetPage(defaultJsonPagination.Page)
	if err != nil {
		return nil, err
	}
	jp.Page = page

	sortBy, err := params.GetSortBy(defaultJsonPagination.SortBy)
	if err != nil {
		return nil, err
	}
	jp.SortBy = sortBy

	order, err := params.GetSortOrder(defaultJsonPagination.Order)
	if err != nil {
		return nil, err
	}
	jp.Order = order

	return jp, nil
}

// StartIndex returns the index number of the first element to be used.
func (p *JsonPagination) StartIndex() int {
	return p.Page * p.PerPage
}

// EndIndex returns the index number of the last element to be used.
func (p *JsonPagination) EndIndex() int {
	return p.StartIndex() + p.PerPage
}

// String returns a URL-style string of the pagination settings.
func (p *JsonPagination) String() string {
	s := fmt.Sprintf("perPage=%d&page=%d&sortBy=%s&order=%s",
		p.PerPage, p.Page, p.SortBy, p.Order)
	return s
}
