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
	"regexp"
	"sort"

	s "github.com/venicegeo/vzutil-versioning/web/app/structs"
	"github.com/venicegeo/vzutil-versioning/web/es"
	u "github.com/venicegeo/vzutil-versioning/web/util"
)

var isSha = regexp.MustCompile(`^[a-f0-9]{40}$`)

type SingleRunner struct {
	app *Application
}

func NewSingleRunner(app *Application) *SingleRunner {
	return &SingleRunner{app}
}

func (sr *SingleRunner) RunAgainstSingle(printHeader string, printLocation chan string, git *s.GitWebhook) *SingleResult {
	explicitSha := isSha.MatchString(git.AfterSha)
	sr.sendStringTo(printLocation, "%sStarting work on %s", printHeader, git.AfterSha)

	var deps []es.Dependency
	var hashes []string
	type SingleReturn struct {
		Name string
		Sha  string
		Deps []string
	}
	dat, err := exec.Command(sr.app.singleLocation, "--all", git.Repository.FullName, git.AfterSha).Output()
	if err != nil {
		sr.sendStringTo(printLocation, "%sUnable to run against %s [%s]", printHeader, git.AfterSha, err.Error())
		return nil
	}
	var singleRet SingleReturn
	if err = json.Unmarshal(dat, &singleRet); err != nil {
		sr.sendStringTo(printLocation, "%sUnable to run against %s [%s]", printHeader, git.AfterSha, err.Error())
		return nil
	}
	if explicitSha && singleRet.Sha != git.AfterSha {
		sr.sendStringTo(printLocation, "%sGeneration failed to run against %s, it ran against sha %s", printHeader, git.AfterSha, singleRet.Sha)
		return nil
	}
	{
		deps = make([]es.Dependency, 0, len(singleRet.Deps))
		for _, d := range singleRet.Deps {
			matches := depRe.FindStringSubmatch(d)
			deps = append(deps, es.Dependency{matches[1], matches[2], matches[3]})
		}
	}
	{
		hashes = make([]string, len(deps))
		for i, d := range deps {
			hash := d.GetHashSum()
			hashes[i] = hash
			exists, err := sr.app.index.ItemExists("dependency", hash)
			if err != nil || !exists {
				go func(dep es.Dependency, h string) {
					resp, err := sr.app.index.PostData("dependency", h, dep)
					if err != nil {
						sr.sendStringTo(printLocation, "%sUnable to create dependency %s [%s]", printHeader, h, err.Error())
					} else if !resp.Created {
						sr.sendStringTo(printLocation, "%sUnable to create dependency %s", printHeader, h)
					}
				}(d, hash)
			}
		}
		sort.Strings(hashes)
	}
	sr.sendStringTo(printLocation, "%sFinished work on %s", printHeader, git.AfterSha)
	return &SingleResult{git.Repository.FullName, git.Repository.Name, singleRet.Sha, git.Ref, deps, hashes, git.Timestamp}
}
func (sr *SingleRunner) sendStringTo(location chan string, format string, args ...interface{}) {
	if location != nil {
		location <- u.Format(format, args...)
	}
}
