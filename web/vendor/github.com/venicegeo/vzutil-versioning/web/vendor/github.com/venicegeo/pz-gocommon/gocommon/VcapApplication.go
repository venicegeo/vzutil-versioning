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

// example:
//     "VCAP_APPLICATION": {
//         "application_id": "14fca253-8087-402e-abf5-8fd40ddda81f",
//         "application_name": "pz-workflow",
//         "application_uris": [
//             "pz-workflow.venicegeo.io"
//         ],
//         "application_version": "5f0ee99d-252c-4f8d-b241-bc3e22534afc",
//         "limits": {
//             "disk": 1024,
//             "fds": 16384,
//             "mem": 512
//         },
//         "name": "pz-workflow",
//         "space_id": "d65a0987-df00-4d69-a50b-657e52cb2f8e",
//         "space_name": "simulator-stage",
//         "uris": [
//             "pz-workflow.venicegeo.io"
//         ],
//         "users": null,
//         "version": "5f0ee99d-252c-4f8d-b241-bc3e22534afc"
//     }

package piazza

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

type VcapApplication struct {
	ApplicationID      string         `json:"application_id"`
	ApplicationName    string         `json:"application_name"`
	ApplicationURIs    []string       `json:"application_uris"`
	ApplicationVersion string         `json:"application_version"`
	Limits             map[string]int `json:"limits"`
	Name               string         `json:"name"`
	SpaceId            string         `json:"space_id"`
	SpaceName          string         `json:"space_name"`
	URIs               []string       `json:"uris"`
	Users              interface{}    `json:"users"` // don't know what the datattype actually is
	Version            string         `json:"version"`

	bindToPort string
	domain     string
}

func NewVcapApplication(serviceName ServiceName) (*VcapApplication, error) {

	var err error
	var vcap *VcapApplication

	str := os.Getenv("VCAP_APPLICATION")
	if str != "" {
		//log.Printf("VCAP_APPLICATION:\n%s", str)
		vcap = &VcapApplication{}

		err = json.Unmarshal([]byte(str), vcap)
		if err != nil {
			return nil, err
		}

		vcap.bindToPort = os.Getenv("PORT")
		if str == "" {
			return nil, errors.New("Unable to read $PORT for PCF deployment")
		}

		vcap.bindToPort = ":" + vcap.bindToPort
		//log.Printf("PORT: %s", vcap.bindToPort)

		full := vcap.GetAddress()
		dot := strings.Index(full, ".")
		if dot == -1 {
			return nil, fmt.Errorf("error extracting domain from address %s", full)
		}
		vcap.domain = full[dot:]

	} else {
		vcap = genLocalVcapApplication(serviceName)
	}

	return vcap, nil
}

func genLocalVcapApplication(serviceName ServiceName) *VcapApplication {
	port, ok := LocalPortNumbers[serviceName]
	if !ok {
		port = "0"
	}
	return &VcapApplication{
		ApplicationName: "myapplicationname",
		ApplicationURIs: []string{"localhost:" + port},
		bindToPort:      "localhost:" + port,
		domain:          ".venicegeo.io",
	}
}

func (vcap *VcapApplication) GetAddress() string {
	return vcap.ApplicationURIs[0]
}

func (vcap *VcapApplication) GetName() ServiceName {
	return ServiceName(vcap.ApplicationName)
}

func (vcap *VcapApplication) GetBindToPort() string {
	return vcap.bindToPort
}

func (vcap *VcapApplication) GetDomain() string {
	return vcap.domain
}
