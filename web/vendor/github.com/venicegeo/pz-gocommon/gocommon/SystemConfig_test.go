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

func Test03SystemConfig(t *testing.T) {
	assert := assert.New(t)

	required := []ServiceName{}

	_, err := NewSystemConfig(PzGoCommon, required)
	assert.NoError(err)
}

func Test04Services(t *testing.T) {
	assert := assert.New(t)

	required := []ServiceName{}

	{
		sys, err := NewSystemConfig(PzGoCommon, required)
		assert.NoError(err)

		actual := sys.GetDomain()
		assert.EqualValues(actual, DefaultDomain)

		addr := "1.2.3.4"
		sys.AddService(PzLogger, addr)

		actual, err = sys.GetAddress(PzLogger)
		assert.NoError(err)
		assert.EqualValues(addr, actual)

		actual, err = sys.GetURL(PzLogger)
		assert.NoError(err)
		assert.EqualValues(actual, "http://"+addr)
	}

	{
		setenvT(t, "DOMAIN", "abc.xyz")
		defer unsetenvT(t, "DOMAIN")

		sys, err := NewSystemConfig(PzGoCommon, required)
		assert.NoError(err)

		actual := sys.GetDomain()
		assert.EqualValues(actual, ".abc.xyz")

		addr := "1.2.3.4"
		sys.AddService(PzLogger, addr)

		actual, err = sys.GetAddress(PzLogger)
		assert.NoError(err)
		assert.EqualValues(addr, actual)

		actual, err = sys.GetURL(PzLogger)
		assert.NoError(err)
		assert.EqualValues(actual, "http://"+addr)
	}
}
