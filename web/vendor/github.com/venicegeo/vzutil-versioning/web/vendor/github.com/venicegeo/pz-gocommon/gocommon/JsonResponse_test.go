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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

//--------------------------

func TestMarshalling(t *testing.T) {
	assert := assert.New(t)

	a := &JsonResponse{
		StatusCode: 10,
	}

	byts, err := json.Marshal(a)
	assert.NoError(err)
	assert.EqualValues("{\"statusCode\":10}", string(byts))

	b := &JsonResponse{}
	err = json.Unmarshal(byts, b)
	assert.NoError(err)
	assert.EqualValues(a, b)

	actual := b.String()
	expected := `{
		StatusCode: 10,
		Data: <nil>,
		Message:
	}`
	assert.Equal(RemoveWhitespace(expected), RemoveWhitespace(actual))
}

func TestIdent(t *testing.T) {
	assert := assert.New(t)

	var id Ident = "01730"
	s := id.String()

	assert.Equal("01730", s)
}

func TestMisc(t *testing.T) {
	assert := assert.New(t)

	a := &JsonResponse{
		StatusCode: 401,
		Message:    "yow",
	}

	assert.True(a.IsError())
	err := a.ToError()
	assert.Equal("{401: yow}", err.Error())

	b := newJsonResponse500(fmt.Errorf("yowyow"))
	assert.True(b.IsError())
	assert.Equal("yowyow", b.Message)

	a.StatusCode = 201
	assert.False(a.IsError())

	assert.Nil(a.ToError())
	assert.Nil(a.SetType())
}
