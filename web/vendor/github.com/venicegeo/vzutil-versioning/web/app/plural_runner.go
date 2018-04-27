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
	u "github.com/venicegeo/vzutil-versioning/web/util"
)

type PluralRunner struct {
	app *Application
}

func NewPluralRunner(app *Application) *PluralRunner {
	return &PluralRunner{app}
}

func (pr *PluralRunner) RunAgainstPluralStr(repos, checkouts []string) (string, error) {
	if len(repos) != len(checkouts) {
		return "", u.Error("Inputs not the same length")
	}
	in := map[string]string{}
	for i, r := range repos {
		in[r] = checkouts[i]
	}
	indat, err := json.Marshal(in)
	if err != nil {
		return "", err
	}
	cmd := exec.Command(pr.app.pluralLocation, "-t", string(indat), "-r", "2")
	dat, err := cmd.Output()
	if err != nil {
		return "", u.Error("%s %s %s", cmd.Args, err, string(dat))
	}
	return string(dat), nil
}
func (pr *PluralRunner) RunAgainstPlural(repos, checkouts []string) (*com.ProjectsDependencies, error) {
	str, err := pr.RunAgainstPluralStr(repos, checkouts)
	if err != nil {
		return nil, err
	}
	var pluralRet com.ProjectsDependencies
	if err = json.Unmarshal([]byte(str), &pluralRet); err != nil {
		return nil, err
	}
	return &pluralRet, nil
}
