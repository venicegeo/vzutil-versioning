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
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/venicegeo/pz-gocommon/gocommon"
)

// MappingElementTypeName is just an alias for a string.
type MappingElementTypeName string

type QueryFormat struct {
	Size  int
	From  int
	Order bool
	Key   string
}

// Constants representing the supported data types for the Event parameters.
const (
	MappingElementTypeString      MappingElementTypeName = "string"
	MappingElementTypeLong        MappingElementTypeName = "long"
	MappingElementTypeInteger     MappingElementTypeName = "integer"
	MappingElementTypeShort       MappingElementTypeName = "short"
	MappingElementTypeByte        MappingElementTypeName = "byte"
	MappingElementTypeDouble      MappingElementTypeName = "double"
	MappingElementTypeFloat       MappingElementTypeName = "float"
	MappingElementTypeDate        MappingElementTypeName = "date"
	MappingElementTypeBool        MappingElementTypeName = "boolean"
	MappingElementTypeBinary      MappingElementTypeName = "binary"
	MappingElementTypeGeoPoint    MappingElementTypeName = "geo_point"
	MappingElementTypeGeoShape    MappingElementTypeName = "geo_shape"
	MappingElementTypeIp          MappingElementTypeName = "ip"
	MappingElementTypeCompletion  MappingElementTypeName = "completion"
	MappingElementTypeStringA     MappingElementTypeName = "[string]"
	MappingElementTypeLongA       MappingElementTypeName = "[long]"
	MappingElementTypeIntegerA    MappingElementTypeName = "[integer]"
	MappingElementTypeShortA      MappingElementTypeName = "[short]"
	MappingElementTypeByteA       MappingElementTypeName = "[byte]"
	MappingElementTypeDoubleA     MappingElementTypeName = "[double]"
	MappingElementTypeFloatA      MappingElementTypeName = "[float]"
	MappingElementTypeDateA       MappingElementTypeName = "[date]"
	MappingElementTypeBoolA       MappingElementTypeName = "[boolean]"
	MappingElementTypeBinaryA     MappingElementTypeName = "[binary]"
	MappingElementTypeGeoPointA   MappingElementTypeName = "[geo_point]"
	MappingElementTypeGeoShapeA   MappingElementTypeName = "[geo_shape]"
	MappingElementTypeIpA         MappingElementTypeName = "[ip]"
	MappingElementTypeCompletionA MappingElementTypeName = "[completion]"
)

// IIndex is an interface to Elasticsearch Index methods
type IIndex interface {
	GetVersion() string

	IndexName() string
	IndexExists() (bool, error)
	TypeExists(typ string) (bool, error)
	ItemExists(typ string, id string) (bool, error)
	Create(settings string) error
	Close() error
	Delete() error
	PostData(typ string, id string, obj interface{}) (*IndexResponse, error)
	PutData(typ string, id string, obj interface{}) (*IndexResponse, error)
	GetByID(typ string, id string) (*GetResult, error)
	DeleteByID(typ string, id string) (*DeleteResponse, error)
	FilterByMatchAll(typ string, format *piazza.JsonPagination) (*SearchResult, error)
	GetAllElements(typ string) (*SearchResult, error)
	FilterByTermQuery(typ string, name string, value interface{}, format *piazza.JsonPagination) (*SearchResult, error)
	FilterByMatchQuery(typ string, name string, value interface{}, format *piazza.JsonPagination) (*SearchResult, error)
	SearchByJSON(typ string, jsn string) (*SearchResult, error)
	SetMapping(typename string, jsn piazza.JsonString) error
	GetTypes() ([]string, error)
	GetMapping(typ string) (interface{}, error)
	AddPercolationQuery(id string, query piazza.JsonString) (*IndexResponse, error)
	DeletePercolationQuery(id string) (*DeleteResponse, error)
	AddPercolationDocument(typ string, doc interface{}) (*PercolateResponse, error)

	DirectAccess(verb string, endpoint string, input interface{}, output interface{}) error
}

// NewIndexInterface constructs an IIndex
func NewIndexInterface(sys *piazza.SystemConfig, index string, settings string, mocking bool) (IIndex, error) {
	var esi IIndex
	var err error

	if mocking {
		esi = NewMockIndex(index)
		return esi, nil
	}

	esi, err = NewIndex(sys, index, settings)
	if err != nil {
		return nil, err
	}

	if esi == nil {
		return nil, errors.New("Index creation failed: returned nil")
	}

	return esi, nil
}

// ConstructMappingSchema takes a map of parameter names to datatypes and
// returns the corresponding ES DSL for it.
func ConstructMappingSchema(name string, items map[string]MappingElementTypeName) (piazza.JsonString, error) {

	const template string = `{
		"%s":{
			"properties":{
				%s
			}
		}
	}`

	stuff := make([]string, len(items))
	i := 0
	for k, v := range items {
		stuff[i] = fmt.Sprintf(`"%s": {"type":"%s"}`, k, v)
		i++
	}

	json := fmt.Sprintf(template, name, strings.Join(stuff, ","))

	return piazza.JsonString(json), nil
}

// NewQueryFormat constructs a QueryFormat
func NewQueryFormat(params *piazza.JsonPagination) *QueryFormat {

	format := &QueryFormat{
		Size:  params.PerPage,
		From:  params.Page * params.PerPage,
		Key:   params.SortBy,
		Order: params.Order == piazza.SortOrderAscending,
	}

	return format
}

type GetData func() (bool, error)

func PollFunction(fn GetData) (bool, error) {
	timeout := time.After(5 * time.Second)
	tick := time.Tick(250 * time.Millisecond)
	for {
		select {
		case <-timeout:
			return false, errors.New("timeout reached")
		case <-tick:
			ok, err := fn()
			if err != nil {
				return false, err
			} else if ok {
				return true, nil
			}
		}
	}
}

func (name MappingElementTypeName) isValidMappingType() bool {
	valid := name.isValidScalarMappingType() ||
		name.isValidArrayMappingType()
	return valid
}

func (name MappingElementTypeName) isValidScalarMappingType() bool {

	switch name {
	case MappingElementTypeString,
		MappingElementTypeLong,
		MappingElementTypeInteger,
		MappingElementTypeShort,
		MappingElementTypeByte,
		MappingElementTypeDouble,
		MappingElementTypeFloat,
		MappingElementTypeDate,
		MappingElementTypeBool,
		MappingElementTypeBinary,
		MappingElementTypeGeoPoint,
		MappingElementTypeGeoShape,
		MappingElementTypeIp,
		MappingElementTypeCompletion:
		return true
	}

	return false
}

func (name MappingElementTypeName) isValidArrayMappingType() bool {

	switch name {
	case MappingElementTypeStringA,
		MappingElementTypeLongA,
		MappingElementTypeIntegerA,
		MappingElementTypeShortA,
		MappingElementTypeByteA,
		MappingElementTypeDoubleA,
		MappingElementTypeFloatA,
		MappingElementTypeDateA,
		MappingElementTypeBoolA,
		MappingElementTypeBinaryA,
		MappingElementTypeGeoPointA,
		MappingElementTypeGeoShapeA,
		MappingElementTypeIpA,
		MappingElementTypeCompletionA:
		return true
	}

	return false
}

func IsValidMappingType(mappingValue interface{}) bool {
	str, ok := mappingValue.(string)
	if !ok {
		return false
	}
	name := MappingElementTypeName(str)

	return name.isValidMappingType()
}

func IsValidArrayTypeMapping(mappingValue interface{}) bool {
	str, ok := mappingValue.(string)
	if !ok {
		return false
	}
	name := MappingElementTypeName(str)

	return name.isValidArrayMappingType()
}
