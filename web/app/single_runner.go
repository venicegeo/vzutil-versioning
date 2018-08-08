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
	"strings"
	"time"

	nt "github.com/venicegeo/pz-gocommon/gocommon"
	c "github.com/venicegeo/vzutil-versioning/common"
	s "github.com/venicegeo/vzutil-versioning/web/app/structs"
	u "github.com/venicegeo/vzutil-versioning/web/util"
)

type SingleRunner struct {
	app            *Application
	isSha          *regexp.Regexp
	findCommitTime *regexp.Regexp
}

func NewSingleRunner(app *Application) *SingleRunner {
	return &SingleRunner{
		app,
		regexp.MustCompile(`^[a-f0-9]{40}$`),
		regexp.MustCompile(`\s*committed\n\s*<relative-time datetime="([^"]+)"`),
	}
}

func (sr *SingleRunner) RunAgainstSingle(printHeader string, printLocation chan string, git *s.GitWebhook) *c.DependencyScan {
	explicitSha := sr.isSha.MatchString(git.AfterSha)
	if !explicitSha {
		panic("FIX THIS")
	}
	sr.sendStringTo(printLocation, "%sStarting work on %s", printHeader, git.AfterSha)

	dat, err := exec.Command(sr.app.singleLocation, "--all", git.Repository.FullName, git.AfterSha).Output()
	if err != nil {
		sr.sendStringTo(printLocation, "%sUnable to run against %s [%s]", printHeader, git.AfterSha, err.Error())
		return nil
	}
	var singleRet c.DependencyScan
	if err = json.Unmarshal(dat, &singleRet); err != nil {
		sr.sendStringTo(printLocation, "%sUnable to run against %s [%s]", printHeader, git.AfterSha, err.Error())
		return nil
	}
	if singleRet.Sha != git.AfterSha {
		sr.sendStringTo(printLocation, "%sGeneration failed to run against %s, it ran against sha %s", printHeader, git.AfterSha, singleRet.Sha)
		return nil
	}
	code, body, _, err := nt.HTTP(nt.GET, "https://github.com/"+singleRet.Fullname+"/commit/"+git.AfterSha, nt.NewHeaderBuilder().GetHeader(), nil)
	if err != nil || code != 200 {
		sr.sendStringTo(printLocation, "%sUnable to find timestamp for %s [%d: %s]", printHeader, singleRet.Fullname, code, err.Error())
		return nil
	}
	matches := sr.findCommitTime.FindStringSubmatch(strings.TrimSpace(string(body)))
	if len(matches) == 1 {
		sr.sendStringTo(printLocation, "%sCould not scrub commit timestamp", printHeader)
		return nil
	}
	if singleRet.Timestamp, err = time.Parse(time.RFC3339, matches[1]); err != nil {
		sr.sendStringTo(printLocation, "%sError parsing timestamp for %s [%s]", printHeader, singleRet.Fullname, err.Error())
		return nil
	}
	return &singleRet
}
func (sr *SingleRunner) sendStringTo(location chan string, format string, args ...interface{}) {
	if location != nil {
		location <- u.Format(format, args...)
	}
}
