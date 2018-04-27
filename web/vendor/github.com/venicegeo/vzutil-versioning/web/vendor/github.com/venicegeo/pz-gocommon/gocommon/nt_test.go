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
	"testing"

	"github.com/stretchr/testify/assert"
)

//--------------------------

func TestNT(t *testing.T) {
	assert := assert.New(t)

	var genericServer *GenericServer
	var server *ThingServer
	var sys *SystemConfig

	{
		var err error
		required := []ServiceName{}
		sys, err = NewSystemConfig(PzGoCommonTest, required)
		if err != nil {
			assert.FailNow(err.Error())
		}
		genericServer = &GenericServer{Sys: sys}
		server = &ThingServer{}
		service := &ThingService{
			assert:  assert,
			IDCount: 0,
			Data:    make(map[string]string),
		}
		server.Init(service)
	}
	{
		var err error
		err = genericServer.Configure(server.routes)
		if err != nil {
			assert.FailNow("server failed to configure: " + err.Error())
		}
		_, err = genericServer.Start()
		if err != nil {
			assert.FailNow("server failed to start: " + err.Error())
		}
	}

	GetValueFromHeader(http.Header{}, "Content-Type")
	_, _, _, err := HTTP(GET, "http://localhost:"+LocalPortNumbers[PzGoCommonTest], NewHeaderBuilder().AddJsonContentType().AddBasicAuth("foo", "bar").GetHeader(), nil)
	if err != nil {
		assert.FailNow(err.Error())
	}

	err = genericServer.Stop()
	assert.NoError(err)

}
