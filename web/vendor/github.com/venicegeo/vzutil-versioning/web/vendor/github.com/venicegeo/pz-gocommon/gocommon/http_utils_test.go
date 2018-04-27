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
	"io/ioutil"
	"os"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

//--------------------------

func TestHttpUtils(t *testing.T) {
	assert := assert.New(t)

	// testing of Http{Get,Post,Put,Delete}Json covered by GenericServer_test.go
	// testing of HTTP{Put,Delete} covered by GenericServer_test.go

	assert.True(!false)
}

func fileExists(s string) bool {
	if _, err := os.Stat(s); os.IsNotExist(err) {
		return false
	}
	return true
}

func TestApiKey(t *testing.T) {
	assert := assert.New(t)

	// will it read from $PZKEY?
	{
		setenvT(t, "PZKEY", "yow")

		key, err := GetApiKey("int")
		assert.NoError(err)
		assert.EqualValues(key, "yow")

		unsetenvT(t, "PZKEY")
	}

	path := os.Getenv("HOME")
	assert.True(path != "")

	path += "/.pzkey"

	// will it read $HOME/.pzkey if $PZKEY not set?
	// (note the test can't control whether $HOME/.pzkey actually exists or not)

	if fileExists(path) {
		key, err := GetApiKey("piazza.venicegeo.io")
		assert.NoError(err)

		raw, err := ioutil.ReadFile(path)
		assert.NoError(err)
		data := map[string]string{}
		err = json.Unmarshal(raw, &data)
		assert.NoError(err)
		actual := data["piazza.venicegeo.io"]

		assert.EqualValues(actual, key)
	} else {
		_, err := GetApiKey("piazza.venicegeo.io")
		assert.Error(err)
	}

	// will it error if the space doesn't exist?
	if fileExists(path) {
		_, err := GetApiKey("la.la.la")
		assert.Error(err)
	}
}

func TestApiServer(t *testing.T) {
	assert := assert.New(t)

	unsetenvT(t, "PZSERVER")

	_, err := GetApiServer()
	assert.Error(err)

	setenvT(t, "PZSERVER", "a.b.c.d")

	pzserver, err := GetApiServer()
	assert.NoError(err)
	assert.EqualValues("a.b.c.d", pzserver)

	unsetenvT(t, "PZSERVER")
}

func TestGetIP(t *testing.T) {
	assert := assert.New(t)
	ip, err := GetExternalIP()
	assert.NoError(err)
	assert.Regexp(regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$`), ip)
}
