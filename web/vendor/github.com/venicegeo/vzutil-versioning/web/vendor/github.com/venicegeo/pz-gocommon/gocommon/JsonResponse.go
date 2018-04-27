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
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
)

//----------------------------------------------------------

// Ident is the type used for representing ID values.
type Ident string

// NoIdent is the "empty" ID value.
const NoIdent Ident = ""

// String returns an id as a string.
func (id Ident) String() string {
	return string(id)
}

//----------------------------------------------------------

// Version captures the information about the version of a service. For now,
// it is just a string.
type Version struct {
	Version string
}

//----------------------------------------------------------

// JsonString is the type used to represent JSON values.
type JsonString string

//----------------------------------------------------------

type JsonResponse struct {
	StatusCode int `json:"statusCode" binding:"required"`

	// only 2xxx -- Data is required (and Type too)
	// Type is a string, taken from the Java model list
	Type       string          `json:"type,omitempty"`
	Data       interface{}     `json:"data,omitempty" binding:"required"`
	Pagination *JsonPagination `json:"pagination,omitempty"` // optional

	// only 4xx and 5xx -- Message is required
	Message string        `json:"message,omitempty"`
	Origin  string        `json:"origin,omitempty"` // optional
	Inner   *JsonResponse `json:"inner,omitempty"`  // optional

	// optional
	Metadata interface{} `json:"metadata,omitempty"`
}

var JsonResponseDataTypes = map[string]string{}

func init() {
	// common types
	JsonResponseDataTypes["string"] = "string"
	JsonResponseDataTypes["[]string"] = "string-list"
	JsonResponseDataTypes["int"] = "int"
	JsonResponseDataTypes["*piazza.Version"] = "version"
}

func (resp *JsonResponse) String() string {
	s := fmt.Sprintf("{StatusCode: %d, Data: %#v, Message: %s}",
		resp.StatusCode, resp.Data, resp.Message)
	return s
}

func (resp *JsonResponse) IsError() bool {
	return resp.StatusCode < 200 || resp.StatusCode > 299
}

func (resp *JsonResponse) ToError() error {
	if !resp.IsError() {
		return nil
	}
	s := fmt.Sprintf("{%d: %s}", resp.StatusCode, resp.Message)
	return errors.New(s)
}

func newJsonResponse500(err error) *JsonResponse {
	return &JsonResponse{StatusCode: http.StatusInternalServerError, Message: err.Error()}
}

func (resp *JsonResponse) SetType() error {
	if resp.Data == nil {
		resp.Type = ""
		return nil
	}

	goName := reflect.TypeOf(resp.Data).String()
	modelName, ok := JsonResponseDataTypes[goName]
	if !ok {
		return fmt.Errorf("Type %s is not a valid response type", goName)
	}
	resp.Type = modelName
	return nil
}

// given a JsonResponse object returned from an http call, and with Data set
// convert it to the given output type
// (formerly called SuperConverter)
func (resp *JsonResponse) ExtractData(output interface{}) error {
	raw, err := json.Marshal(resp.Data)
	if err != nil {
		return err
	}

	return json.Unmarshal(raw, output)
}
