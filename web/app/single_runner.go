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
	h "github.com/venicegeo/vzutil-versioning/common/history"
	"github.com/venicegeo/vzutil-versioning/web/es/types"
	u "github.com/venicegeo/vzutil-versioning/web/util"
)

type SingleRunner struct {
	app            *Application
	findCommitTime *regexp.Regexp
}

type SingleRunnerRequest struct {
	repository *Repository
	sha        string
	ref        string
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

func (sr *SingleRunner) RunAgainstSingle(printHeader string, printLocation chan string, request *SingleRunnerRequest) *types.Scan {
	sr.sendStringTo(printLocation, "%sStarting work on %s", printHeader, request.sha)

	args := make([]string, len(request.repository.DependencyInfo.FilesToScan)*2, len(request.repository.DependencyInfo.FilesToScan)*2)
	i := 0
	for _, f := range request.repository.DependencyInfo.FilesToScan {
		args[i] = "--f"
		args[i+1] = strings.TrimPrefix(f, request.repository.DependencyInfo.RepoFullname)[1:]
		i += 2
	}
	args = append(args, request.repository.DependencyInfo.RepoFullname)
	switch request.repository.DependencyInfo.CheckoutType {
	case types.IncomingSha:
		args = append(args, request.sha)
	case types.ExactSha:
		args = append(args, request.repository.DependencyInfo.CustomField)
	case types.CustomRef:
		args = append(args, request.repository.DependencyInfo.CustomField)
	case types.SameRef:
		args = append(args, request.ref)
	}

	dat, err := exec.Command(sr.app.singleLocation, args...).CombinedOutput()
	if err != nil {
		sr.sendStringTo(printLocation, "%sUnable to run against %s [%s]\n%s", printHeader, request.sha, err.Error(), string(dat))
		return nil
	}
	res := &types.Scan{
		RepoFullname: request.repository.Fullname,
		ProjectId:    request.repository.ProjectId,
		Refs:         []string{request.ref},
		Sha:          request.sha,
	}
	var singleRet = new(c.DependencyScan)
	if err = json.Unmarshal(dat, singleRet); err != nil {
		sr.sendStringTo(printLocation, "%sUnable to run against %s [%s]", printHeader, request.sha, err.Error())
		return nil
	}
	//TODO
	//	if singleRet.Sha != request.sha {
	//		sr.sendStringTo(printLocation, "%sGeneration failed to run against %s, it ran against sha %s", printHeader, request.sha, singleRet.Sha)
	//		return nil
	//	}
	{ //Find timestamp of commit
		code, body, _, err := nt.HTTP(nt.GET, "https://github.com/"+request.repository.Fullname+"/commit/"+request.sha, nt.NewHeaderBuilder().GetHeader(), nil)
		if err != nil || code != 200 {
			sr.sendStringTo(printLocation, "%sUnable to find timestamp for %s [%d: %s]", printHeader, request.repository.Fullname, code, err.Error())
			return nil
		}
		matches := sr.findCommitTime.FindStringSubmatch(strings.TrimSpace(string(body)))
		if len(matches) == 1 {
			sr.sendStringTo(printLocation, "%sCould not scrub commit timestamp", printHeader)
			return nil
		}
		if res.Timestamp, err = time.Parse(time.RFC3339, matches[1]); err != nil {
			sr.sendStringTo(printLocation, "%sError parsing timestamp for %s [%s]", printHeader, request.repository.Fullname, err.Error())
			return nil
		}
	}
	sr.sendStringTo(printLocation, "%sFinished work on %s", printHeader, request.sha)
	res.Scan = singleRet
	return res
}
func (sr *SingleRunner) sendStringTo(location chan string, format string, args ...interface{}) {
	if location != nil {
		location <- u.Format(format, args...)
	}
}

func (sr *SingleRunner) History(repo string) (h.HistoryTree, error) {
	dat, err := exec.Command(sr.app.singleLocation, "-history", repo).CombinedOutput()
	if err != nil {
		return nil, err
	}
	var tree h.HistoryTree
	return tree, json.Unmarshal(dat, &tree)
}
