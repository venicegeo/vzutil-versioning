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
	u "github.com/venicegeo/vzutil-versioning/web/util"
)

type SingleRunner struct {
	app            *Application
	findCommitTime *regexp.Regexp
}

type SingleRunnerRequest struct {
	Fullname  string
	Sha       string
	Ref       string
	Requester string
	Files     []string
}

func NewSingleRunner(app *Application) *SingleRunner {
	return &SingleRunner{
		app,
		regexp.MustCompile(`\s*committed\n\s*<relative-time datetime="([^"]+)"`),
	}
}

func (sr *SingleRunner) ScanWithSingle(fullName string) ([]string, error) {
	dat, err := exec.Command(sr.app.singleLocation, "--scan", fullName, "master").Output()
	if err != nil {
		return nil, err
	}
	var output struct {
		Files []string `json:"files"`
	}
	if err = json.Unmarshal(dat, &output); err != nil {
		return nil, err
	}
	for i, f := range output.Files {
		output.Files[i] = u.Format("%s/%s", fullName, f)
	}
	return output.Files, nil
}

func (sr *SingleRunner) RunAgainstSingle(printHeader string, printLocation chan string, request *SingleRunnerRequest) *c.DependencyScan {
	sr.sendStringTo(printLocation, "%sStarting work on %s", printHeader, request.Sha)

	args := make([]string, len(request.Files)*2, len(request.Files)*2)
	i := 0
	for _, f := range request.Files {
		args[i] = "--f"
		args[i+1] = strings.TrimPrefix(f, request.Fullname)[1:]
		i += 2
	}
	args = append(args, "--requester", request.Requester, request.Fullname, request.Sha)

	dat, err := exec.Command(sr.app.singleLocation, args...).Output()
	if err != nil {
		sr.sendStringTo(printLocation, "%sUnable to run against %s [%s]", printHeader, request.Sha, err.Error())
		return nil
	}
	var singleRet c.DependencyScan
	if err = json.Unmarshal(dat, &singleRet); err != nil {
		sr.sendStringTo(printLocation, "%sUnable to run against %s [%s]", printHeader, request.Sha, err.Error())
		return nil
	}
	if singleRet.Sha != request.Sha {
		sr.sendStringTo(printLocation, "%sGeneration failed to run against %s, it ran against sha %s", printHeader, request.Sha, singleRet.Sha)
		return nil
	}
	code, body, _, err := nt.HTTP(nt.GET, "https://github.com/"+singleRet.Fullname+"/commit/"+request.Sha, nt.NewHeaderBuilder().GetHeader(), nil)
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
