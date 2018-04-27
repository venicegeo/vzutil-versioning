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
	"fmt"
	"net/http"
	"os"
	"strings"
)

const DefaultElasticsearchAddress = "localhost:9200"
const DefaultKafkaAddress = "localhost:9092"
const DefaultPzLoggerAddress = "localhost:14600"
const DefaultPzUuidgenAddress = "localhost:14800"
const DefaultDomain = ".venicegeo.io"
const DefaultProtocol = "http"

const waitTimeoutMs = 3000
const waitSleepMs = 250

type ServiceName string

const (
	PzDiscover          ServiceName = "pz-discover"
	PzElasticSearch     ServiceName = "pz-elasticsearch"
	PzGoCommon          ServiceName = "PZ-GOCOMMON"     // not a real service, just for testing
	PzGoCommonTest      ServiceName = "PZ-GOCOMMONTEST" // not a real service, just for testing
	PzKafka             ServiceName = "pz-kafka"
	PzLogger            ServiceName = "pz-logger"
	PzUuidgen           ServiceName = "pz-uuidgen"
	PzWorkflow          ServiceName = "pz-workflow"
	PzsvcHello          ServiceName = "pzsvc-hello"
	PzServiceController ServiceName = "pz-servicecontroller"
	PzMetrics           ServiceName = "pz-metrics"
	PzIdam              ServiceName = "pz-idam"
)

var LocalPortNumbers = map[ServiceName]string{
	PzWorkflow:          "20000",
	PzLogger:            "20001",
	PzUuidgen:           "20002",
	PzDiscover:          "20003",
	PzElasticSearch:     "20004",
	PzServiceController: "20005",
	PzMetrics:           "20006",
	PzKafka:             "20007",
	PzsvcHello:          "20008",
	PzGoCommon:          "20009",
	PzGoCommonTest:      "20010",
	PzIdam:              "20011",
}

var EndpointPrefixes = map[ServiceName]string{
	PzDiscover:          "",
	PzElasticSearch:     "",
	PzKafka:             "",
	PzLogger:            "",
	PzUuidgen:           "",
	PzWorkflow:          "",
	PzsvcHello:          "",
	PzServiceController: "",
	PzMetrics:           "",
	PzIdam:              "",
}

var HealthcheckEndpoints = map[ServiceName]string{
	PzDiscover:          "",
	PzElasticSearch:     "",
	PzKafka:             "",
	PzLogger:            "/",
	PzUuidgen:           "/",
	PzWorkflow:          "/",
	PzsvcHello:          "/",
	PzServiceController: "",
	PzMetrics:           "/",
	PzIdam:              "",
}

type ServicesMap map[ServiceName]string

type SystemConfig struct {
	// our own service
	Name    ServiceName
	Address string
	BindTo  string

	// our external services
	endpoints ServicesMap

	Space string // int or stage or prod or...
	PiazzaSystem string // System-level username

	vcapApplication *VcapApplication
	vcapServices    *VcapServices
	domain          string
}

func NewSystemConfig(serviceName ServiceName,
	requiredServices []ServiceName) (*SystemConfig, error) {

	var err error

	sys := &SystemConfig{endpoints: make(ServicesMap)}

	sys.vcapApplication, err = NewVcapApplication(serviceName)
	if err != nil {
		return nil, err
	}

	sys.vcapServices, err = NewVcapServices()
	if err != nil {
		return nil, err
	}

	if sys.vcapApplication != nil {
		sys.domain = sys.vcapApplication.GetDomain()
	} else {
		sys.domain = DefaultDomain
	}

	if os.Getenv("DOMAIN") != "" {
		sys.domain = os.Getenv("DOMAIN")
		if !strings.HasPrefix(sys.domain, ".") {
			sys.domain = "." + sys.domain
		}
	}

	sys.Space = os.Getenv("SPACE")
	if sys.Space == "" {
		sys.Space = "int"
	}

	sys.PiazzaSystem = os.Getenv("PIAZZA_SYSTEM")
	if sys.PiazzaSystem == "" {
		sys.PiazzaSystem = "piazzaSystem"
	}

	// set some data about our own service first
	sys.Name = serviceName
	sys.Address = sys.vcapApplication.GetAddress()
	sys.BindTo = sys.vcapApplication.GetBindToPort()

	// set the services table with the services we require,
	// using VcapServices to get the addresses
	err = sys.checkRequirements(requiredServices)
	if err != nil {
		return nil, err
	}

	err = sys.runHealthChecks()
	if err != nil {
		return nil, err
	}

	return sys, nil
}

func (sys *SystemConfig) checkRequirements(requirements []ServiceName) error {

	for _, name := range requirements {

		if name == sys.Name {
			sys.AddService(name, sys.Address)

		} else {
			if addr, ok := sys.vcapServices.Services[name]; !ok {
				// the service we want is not in VCAP, so fake it
				sys.AddService(name, string(name)+sys.domain)

			} else {
				// the service we want is in VCAP, with a full and valid address
				sys.AddService(name, addr)
			}
		}
	}

	return nil
}

func (sys *SystemConfig) runHealthChecks() error {
	for name, addr := range sys.endpoints {
		if name == sys.Name || name == PzKafka {
			continue
		}

		url := fmt.Sprintf("%s://%s%s", DefaultProtocol, addr, HealthcheckEndpoints[name])

		resp, err := http.Get(url)
		if err != nil {
			return fmt.Errorf("Health check errored for service: %s at %s <%#v>", name, url, resp)
		}

		err = resp.Body.Close()
		if err != nil {
			return err
		}

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("Health check failed for service: %s at %s <%#v>", name, url, resp)
		}
	}

	return nil
}

// it is explicitly allowed for outsiders to update an existing service, but we'll log it just to be safe
func (sys *SystemConfig) AddService(name ServiceName, address string) {
	sys.endpoints[name] = address
}

func (sys *SystemConfig) GetAddress(name ServiceName) (string, error) {
	addr, ok := sys.endpoints[name]
	if !ok {
		return "", fmt.Errorf("Unknown service: %s", name)
	}

	return addr, nil
}

func (sys *SystemConfig) GetURL(name ServiceName) (string, error) {
	addr, err := sys.GetAddress(name)
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("%s://%s%s", DefaultProtocol, addr, EndpointPrefixes[name])

	return url, nil
}

func (sys *SystemConfig) GetDomain() string {
	return sys.domain
}

func (sys *SystemConfig) WaitForService(name ServiceName) error {
	addr, err := sys.GetAddress(name)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s://%s", DefaultProtocol, addr)
	return WaitForService(name, url)
}

func (sys *SystemConfig) WaitForServiceToDie(name ServiceName) error {
	addr, err := sys.GetAddress(name)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s://%s", DefaultProtocol, addr)
	return WaitForServiceToDie(name, url)
}
