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

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

//----------------------------------------------------------

// HttpQueryParams holds a list of the query parameters of a URL, e.g.
//     ...?count=8&foo=bar
//
// We don't want to pass http.Request objects into the Services classes
// (and certainly not a gin.Context!), and we need this kind of information
// a lot, so we'll keep a special data structure which actually understands
// the semantics as well as the syntax.
type HttpQueryParams struct {
	raw map[string]string
}

// NewQueryParams creates an HttpQueryParams object from an http.Request.
func NewQueryParams(request *http.Request) *HttpQueryParams {
	params := HttpQueryParams{raw: make(map[string]string)}

	for k, v := range request.URL.Query() {
		params.raw[k] = v[0]
	}
	return &params
}

// AddString adds a key and string value to the parameter set, e.g. "?foo=bar".
func (params *HttpQueryParams) AddString(key string, value string) {
	if params.raw == nil {
		params.raw = make(map[string]string)
	}
	params.raw[key] = value
}

// AddTime adds a key and time value to the parameter set.
func (params *HttpQueryParams) AddTime(key string, value time.Time) {
	if params.raw == nil {
		params.raw = make(map[string]string)
	}
	params.raw[key] = value.Format(time.RFC3339)
}

// GetAsInt retrieves the (string) value of a key from parameter set and returns it as an int.
func (params *HttpQueryParams) GetAsInt(key string, defalt int) (int, error) {
	if key == "" {
		return defalt, nil
	}

	value, ok := params.raw[key]
	if !ok || value == "" {
		return defalt, nil
	}

	i, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}

	return i, nil
}

// GetAsID retrieves the (string) value of a key from parameter set and
// returns it as an Ident.
func (params *HttpQueryParams) GetAsID(key string, defalt string) (Ident, error) {
	str, err := params.GetAsString(key, defalt)
	return Ident(str), err
}

// GetAsString retrieves the (string) value of a key from parameter set and returns it.
func (params *HttpQueryParams) GetAsString(key string, defalt string) (string, error) {
	if key == "" {
		return defalt, nil
	}

	value, ok := params.raw[key]
	if !ok || value == "" {
		return defalt, nil
	}

	s := value
	return s, nil
}

// GetAsSortOrder retrieves the (string) value of a key from parameter set
// and returns it as SortOrder value.
func (params *HttpQueryParams) GetAsSortOrder(key string, defalt SortOrder) (SortOrder, error) {
	if key == "" {
		return defalt, nil
	}

	value, ok := params.raw[key]
	if !ok || value == "" {
		return defalt, nil
	}

	var order SortOrder
	switch strings.ToLower(value) {
	case "desc":
		order = SortOrderDescending
	case "asc":
		order = SortOrderAscending
	default:
		return "",
			fmt.Errorf("query argument for \"%s\" must be \"asc\" or \"desc\"", value)
	}

	return order, nil
}

// GetAsTime retrieves the (string) value of a key from parameter set and
// returns it as time value.
func (params *HttpQueryParams) GetAsTime(key string, defalt time.Time) (time.Time, error) {
	if key == "" {
		return defalt, nil
	}

	value, ok := params.raw[key]
	if !ok || value == "" {
		return defalt, nil
	}

	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, err
	}

	return t, nil
}

// GetAfter retrieves the value of the "after" parameter.
func (params *HttpQueryParams) GetAfter(defalt time.Time) (time.Time, error) {
	return params.GetAsTime("after", defalt)
}

// GetBefore retrieves the value of the "before" parameter.
func (params *HttpQueryParams) GetBefore(defalt time.Time) (time.Time, error) {
	return params.GetAsTime("before", defalt)
}

// GetCount retrieves the value of the "count" parameter.
func (params *HttpQueryParams) GetCount(defalt int) (int, error) {
	return params.GetAsInt("count", defalt)
}

// GetSortOrder retrieves the value of the "order" parameter.
func (params *HttpQueryParams) GetSortOrder(defalt SortOrder) (SortOrder, error) {
	return params.GetAsSortOrder("order", defalt)
}

// GetPage retrieves the value of the "page" parameter.
func (params *HttpQueryParams) GetPage(defalt int) (int, error) {
	return params.GetAsInt("page", defalt)
}

// GetPerPage retrieves the value of the "perPage" parameter.
func (params *HttpQueryParams) GetPerPage(defalt int) (int, error) {
	return params.GetAsInt("perPage", defalt)
}

// GetSortBy retrieves the value of the "sortBy" parameter.
func (params *HttpQueryParams) GetSortBy(defalt string) (string, error) {
	return params.GetAsString("sortBy", defalt)
}

// String returns the parameter list expressed in URL style.
func (params *HttpQueryParams) String() string {

	if params == nil || params.raw == nil {
		return ""
	}

	s := ""

	for key, value := range params.raw {
		if s != "" {
			s += "&"
		}
		s += fmt.Sprintf("%s=%s", key, value)
	}
	return s
}
