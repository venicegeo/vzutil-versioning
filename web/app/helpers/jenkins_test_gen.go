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
package helpers

import (
	"bytes"

	u "github.com/venicegeo/vzutil-versioning/web/util"
)

type Testing_Pipeline struct {
	displayName string
	basicUrl    string
	repoName    string
	builds      []*Testing_Build
}
type Testing_Build struct {
	repoUrl   string
	timestamp uint64
	sha       string
	stages    []*Testing_Stage
}
type Testing_Stage struct {
	name    string
	success bool
	targets []*Testing_Target
}
type Testing_Target struct {
	org    string
	space  string
	pushed bool
}

func NewPipeline(displayName, repoName, jenkinsUrl string) *Testing_Pipeline {
	return &Testing_Pipeline{displayName, jenkinsUrl + "/job/" + displayName, repoName, []*Testing_Build{}}
}

func (t *Testing_Pipeline) GetRepoUrl() string {
	return "https://github.com/" + t.repoName
}

func (t *Testing_Pipeline) GetUrl() string {
	return t.basicUrl
}

func (t *Testing_Pipeline) AddBuild(build *Testing_Build) *Testing_Pipeline {
	build.repoUrl = t.GetRepoUrl()
	t.builds = append(t.builds, build)
	return t
}

func NewBuild(timestamp uint64, sha string) *Testing_Build {
	return &Testing_Build{"", timestamp, sha, []*Testing_Stage{}}
}

func (t *Testing_Build) AddStage(stage *Testing_Stage) *Testing_Build {
	t.stages = append(t.stages, stage)
	return t
}

func (t *Testing_Stage) AddTarget(target *Testing_Target) *Testing_Stage {
	t.targets = append(t.targets, target)
	return t
}

func NewTarget(org, space string, pushed bool) *Testing_Target {
	return &Testing_Target{org, space, pushed}
}

func NewStage(name string, success bool) *Testing_Stage {
	return &Testing_Stage{name, success, []*Testing_Target{}}
}

func (t *Testing_Build) ConsoleText() string {
	buf := bytes.NewBufferString("")
	buf.WriteString(u.Format("Cloning repository %s\n", t.repoUrl))
	buf.WriteString(u.Format(" > git checkout -b somebranch %s\n", t.sha))
	for _, stage := range t.stages {
		buf.WriteString("[Pipeline] stage\n")
		buf.WriteString(u.Format("[Pipeline] { (%s)\n", stage.name))
		for _, target := range stage.targets {
			buf.WriteString(u.Format("+ cf target -o %s -s %s\n", target.org, target.space))
			if target.pushed {
				buf.WriteString("cf push\n")
			}
		}
		buf.WriteString("[Pipeline] // stage\n")
		if !stage.success {
			buf.WriteString("Finished: FAILURE")
		}
	}
	return buf.String()
}

func (t *Testing_Build) Api() string {
	return u.Format(`{"timestamp":%v}`, t.timestamp)
}

func (t *Testing_Pipeline) Api() string {
	buf := bytes.NewBufferString(`{
	"builds": [
`)
	for i := len(t.builds) - 1; i >= 0; i-- {
		buf.WriteString(u.Format(`{"number":%d}`, i))
		if i != 0 {
			buf.WriteString(",")
		}
		buf.WriteString("\n")
	}
	buf.WriteString("]}")
	return buf.String()
}

func (t *Testing_Pipeline) CreateHttpMap() u.HTTP {
	res := u.NewMap()
	res.Add(t.basicUrl+"/api/json", 200, t.Api())
	for i, build := range t.builds {
		res.Add(u.Format("%s/%d/api/json", t.basicUrl, i), 200, build.Api())
		res.Add(u.Format("%s/%d/consoleText", t.basicUrl, i), 200, build.ConsoleText())
	}
	return res
}
