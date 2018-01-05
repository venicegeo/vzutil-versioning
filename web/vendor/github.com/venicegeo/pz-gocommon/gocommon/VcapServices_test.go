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

func Test06VcapServices(t *testing.T) {
	assert := assert.New(t)

	unsetenvT(t, "VCAP_SERVICES")

	vcap, err := NewVcapServices()
	assert.NoError(err)

	assert.EqualValues("localhost:9092", vcap.Services["pz-kafka"])
	assert.EqualValues("pz-kafka", vcap.UserProvided[1].Name)

	env :=
		`{
			"user-provided": [
				{
					"credentials": {
						"host": "172.32.125.109:9200"
					},
					"label": "user-provided",
					"name": "pz-elasticsearch",
					"syslog_drain_url": "",
					"tags": []
				}
				]
			}`

	setenvT(t, "VCAP_SERVICES", env)
	defer unsetenvT(t, "VCAP_SERVICES")

	vcap, err = NewVcapServices()
	assert.NoError(err)

	assert.EqualValues("172.32.125.109:9200", vcap.Services["pz-elasticsearch"])
	assert.EqualValues("pz-elasticsearch", vcap.UserProvided[0].Name)
}
