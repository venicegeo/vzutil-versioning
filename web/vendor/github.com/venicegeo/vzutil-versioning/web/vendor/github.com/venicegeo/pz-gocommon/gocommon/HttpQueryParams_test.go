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
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

//--------------------------

func TestQueryParams(t *testing.T) {
	assert := assert.New(t)

	addr, err := url.Parse("http://example.com/index.html?a=1&b=foo&c=&d=4")
	assert.NoError(err)

	req := http.Request{URL: addr}

	params := NewQueryParams(&req)

	str := "a=1&b=foo&c=&d=4"
	assert.Equal(len(str), len(params.String()))
	assert.True(strings.Contains(params.String(), "a=1"))
	assert.True(strings.Contains(params.String(), "b=foo"))
	assert.True(strings.Contains(params.String(), "c="))
	assert.True(strings.Contains(params.String(), "d=4"))

	a, err := params.GetAsInt("a", 0)
	assert.NoError(err)
	assert.NotNil(a)
	assert.Equal(1, a)

	b, err := params.GetAsString("b", "")
	assert.NoError(err)
	assert.NotNil(b)
	assert.EqualValues("foo", b)

	bb, err := params.GetAsString("bb", "")
	assert.NoError(err)
	assert.Empty(bb)

	bbb, err := params.GetAsString("bbb", "bar")
	assert.NoError(err)
	assert.NotNil(bbb)
	assert.EqualValues("bar", bbb)

	c, err := params.GetAsInt("c", 0)
	assert.NoError(err)
	assert.Zero(c)

	cc, err := params.GetAsInt("c", 7)
	assert.NoError(err)
	assert.NotNil(cc)
	assert.EqualValues(7, cc)

	var s string

	params.AddString("stringkey", "asdf")
	s, err = params.GetAsString("stringkey", "")
	assert.NoError(err)
	assert.Equal("asdf", s)

	s, err = params.GetAsString("notstringkey", "Foo!")
	assert.NoError(err)
	assert.Equal("Foo!", s)

	id, err := params.GetAsID("stringkey", "")
	assert.NoError(err)
	assert.EqualValues(Ident("asdf"), id)

	tim := time.Now()
	params.AddTime("timekey", tim)
	tim2, err := params.GetAsTime("timekey", time.Time{})
	assert.NoError(err)
	assert.Equal(tim.Second(), tim2.Second())

	params.AddTime("before", tim)
	tim3, err := params.GetBefore(time.Time{})
	assert.NoError(err)
	assert.Equal(tim.Second(), tim3.Second())

	params.AddTime("after", tim)
	tim4, err := params.GetAfter(time.Time{})
	assert.NoError(err)
	assert.Equal(tim.Second(), tim4.Second())

	addr, err = url.Parse("http://example.com/index.html?count=19")
	assert.NoError(err)
	req = http.Request{URL: addr}
	params = NewQueryParams(&req)
	ii, err := params.GetCount(0)
	assert.NoError(err)
	assert.Equal(19, ii)

	params.raw = nil
	params.AddString("stringkey1", "asdf")
	s, err = params.GetAsString("stringkey1", "")
	assert.NoError(err)
	assert.Equal("asdf", s)

	params.raw = nil
	tim = time.Now()
	params.AddTime("timekey1", tim)
	tim2, err = params.GetAsTime("timekey1", time.Time{})
	assert.NoError(err)
	assert.Equal(tim.Second(), tim2.Second())

	_, err = params.GetAsInt("", 1)
	assert.NoError(err)
	_, err = params.GetAsString("", "")
	assert.NoError(err)

	params.AddString("sortie", "frobnitz")
	so, err := params.GetAsSortOrder("", SortOrderAscending)
	assert.NoError(err)
	assert.Equal(SortOrderAscending, so)
	_, err = params.GetAsSortOrder("sortie", SortOrderAscending)
	assert.Error(err)
	params.AddString("sortie", "desc")
	so, err = params.GetAsSortOrder("sortie", SortOrderAscending)
	assert.NoError(err)
	assert.Equal(SortOrderDescending, so)

	_, err = params.GetAsTime("", time.Now())
	assert.NoError(err)
	_, err = params.GetAsTime("notime", time.Now())
	assert.NoError(err)

	params.raw = nil
	assert.Equal("", params.String())
}
