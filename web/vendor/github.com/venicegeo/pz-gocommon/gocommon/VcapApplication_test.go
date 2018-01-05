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
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

//--------------------------

func unsetenvT(t *testing.T, v string) {
	assert := assert.New(t)
	err := os.Unsetenv(v)
	assert.NoError(err)
}

func setenvT(t *testing.T, k string, v string) {
	assert := assert.New(t)
	err := os.Setenv(k, v)
	assert.NoError(err)
}

func Test05VcapApplication(t *testing.T) {
	assert := assert.New(t)

	unsetenvT(t, "VCAP_APPLICATION")
	unsetenvT(t, "PORT")

	vcap, err := NewVcapApplication(PzWorkflow)
	assert.NoError(err)

	assert.EqualValues("localhost:20000", vcap.GetAddress())
	assert.EqualValues("localhost:20000", vcap.GetBindToPort())
	assert.EqualValues("myapplicationname", vcap.GetName())

	env :=
		`{
         "application_id": "14fca253-8087-402e-abf5-8fd40ddda81f",
         "application_name": "pz-workflow",
         "application_uris": [
             "pz-workflow.venicegeo.io"
         ],
         "application_version": "5f0ee99d-252c-4f8d-b241-bc3e22534afc",
         "limits": {
             "disk": 1024,
             "fds": 16384,
             "mem": 512
         },
         "name": "pz-workflow",
         "space_id": "d65a0987-df00-4d69-a50b-657e52cb2f8e",
         "space_name": "simulator-stage",
         "uris": [
             "pz-workflow.venicegeo.io"
         ],
         "users": null,
         "version": "5f0ee99d-252c-4f8d-b241-bc3e22534afc"
     }
`
	setenvT(t, "VCAP_APPLICATION", env)
	defer unsetenvT(t, "VCAP_APPLICATION")
	setenvT(t, "PORT", "6280")
	defer unsetenvT(t, "PORT")

	vcap, err = NewVcapApplication(PzWorkflow)
	assert.NoError(err)

	assert.EqualValues("pz-workflow"+DefaultDomain, vcap.GetAddress())
	assert.EqualValues(":6280", vcap.GetBindToPort())
	assert.EqualValues("pz-workflow", vcap.GetName())
}
