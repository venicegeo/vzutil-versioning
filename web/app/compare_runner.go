// Copyright 2018, RadiantBlue Technologies, Inc.
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

package app

import (
	"encoding/json"
	"os/exec"

	com "github.com/venicegeo/vzutil-versioning/common"
)

type CompareRunner struct {
	app *Application
}

func NewCompareRunner(app *Application) *CompareRunner {
	return &CompareRunner{app}
}

func (cr *CompareRunner) CompareStrings(actual, expected string) (string, error) {
	res, err := exec.Command(cr.app.compareLocation, "-as", actual, "-es", expected).Output()
	return string(res), err
}

func (cr *CompareRunner) CompareProjects(actual, expected com.ProjectsDependencies) (string, error) {
	var adat, bdat []byte
	var err error
	if adat, err = json.Marshal(actual); err != nil {
		return "", err
	}
	if bdat, err = json.Marshal(expected); err != nil {
		return "", err
	}
	return cr.CompareStrings(string(adat), string(bdat))
}
