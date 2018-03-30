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

func (pr *PluralRunner) RunAgainstPlural(repos, checkouts []string) (*com.ProjectDependencies, error) {
	if len(repos) != len(checkouts) {
		return nil, u.Error("Inputs not the same length")
	}
	in := map[string]string{}
	for i, r := range repos {
		in[r] = checkouts[i]
	}
	indat, err := json.Marshal(in)
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(pr.app.pluralLocation, "-t", string(indat))
	dat, err := cmd.Output()
	if err != nil {
		return nil, u.Error("%s %s %s", cmd.Args, err, string(dat))
	}
	var pluralRet com.ProjectDependencies
	if err = json.Unmarshal(dat, &pluralRet); err != nil {
		return nil, err
	}
	return &pluralRet, nil
}
