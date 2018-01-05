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
	"testing"

	"github.com/stretchr/testify/assert"
)

//--------------------------

func TestUtil(t *testing.T) {
	assert := assert.New(t)

	assert.True(!false)
}

func TestStructStringToInterface(t *testing.T) {
	assert := assert.New(t)

	s := `{ "f": "v" }`
	i, err := StructStringToInterface(s)
	assert.NoError(err)

	m := i.(map[string]interface{})
	assert.Equal("v", m["f"])
}

func TestStructInterfaceToString(t *testing.T) {
	assert := assert.New(t)

	m := map[string]interface{}{"f": "v"}
	s, err := StructInterfaceToString(m)
	assert.NoError(err)

	expected := `{ "f": "v" }`
	assert.NotEqual(expected, s)
	assert.Equal(RemoveWhitespace(expected), RemoveWhitespace(s))
}

func TestGetVarsFromStruct(t *testing.T) {
	assert := assert.New(t)

	m := map[string]interface{}{
		"f": "v",
		"g": map[string]interface{}{"a": "b"},
	}
	r, err := GetVarsFromStruct(m)
	assert.NoError(err)

	assert.Len(r, 2)
	assert.Equal("v", r["f"])
	assert.Equal("b", r["g.a"])

	_, err = GetVarsFromStruct(5)
	assert.Error(err)
}

func TestValueIsValidArray(t *testing.T) {
	assert := assert.New(t)

	m := map[string]interface{}{
		"f": "v",
	}
	a := []int{1, 3, 5, 7}
	s := "asdf"

	assert.False(ValueIsValidArray(5))
	assert.False(ValueIsValidArray(s))
	assert.False(ValueIsValidArray(m))
	assert.False(ValueIsValidArray(s[1:2]))

	assert.True(ValueIsValidArray(a))
}

func TestCharAt(t *testing.T) {
	assert := assert.New(t)

	assert.NotEqual("s", CharAt("asdf", 0))
	assert.Equal("s", CharAt("asdf", 1))
}

func TestRemoveWhitespace(t *testing.T) {
	assert := assert.New(t)

	assert.Equal("asdf", RemoveWhitespace(" a sd f  \t "))
	assert.Equal("asdf", RemoveWhitespace("asdf"))
}
func TestInsertString(t *testing.T) {
	assert := assert.New(t)

	assert.Equal("asdf", InsertString("af", "sd", 1))
}

func TestSplitString(t *testing.T) {
	assert := assert.New(t)

	p, q := SplitString("asdf", 2)
	assert.Equal("as", p)
	assert.Equal("df", q)
}
