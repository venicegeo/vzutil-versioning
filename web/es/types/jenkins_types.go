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

package types

import (
	"strings"
)

func DotJoin(strs ...string) string {
	return strings.Join(strs, ".")
}

//--------------------------------------------------------------------------------

type JenkinsPipeline struct {
	Id           string   `json:"id"`
	ProjectId    string   `json:"project_id"`    //The project the repo is in
	RepositoryId string   `json:"repository_id"` //The repository org/repo
	PipelineInfo []string `json:"pipeline_info"` //The job parts from the jenkins url
}

const (
	JenkinsPipeline_QField_Id           = `id`
	JenkinsPipeline_QField_ProjectId    = `project_id`
	JenkinsPipeline_QField_RepositoryId = `repository_id`
	JenkinsPipeline_QField_PipelineInfo = `pipeline_info`
)

const JenkinsPipeline_QMapping = `{
	"dynamic":"strict",
	"properties":{
		"` + JenkinsPipeline_QField_Id + `":{"type":"keyword"},
		"` + JenkinsPipeline_QField_ProjectId + `":{"type":"keyword"},
		"` + JenkinsPipeline_QField_RepositoryId + `":{"type":"keyword"},
		"` + JenkinsPipeline_QField_PipelineInfo + `":{"type":"keyword"}
	}
}`

//--------------------------------------------------------------------------------

type JenkinsBuildStage struct {
	Name    string `json:"name"`
	Success bool   `json:"success"`
}

const (
	JenkinsBuildStage_QField_Name    = `name`
	JenkinsBuildStage_QField_Success = `success`
)

const JenkinsBuildStage_QMapping = `{
	"dynamic":"strict",
	"properties": {
		"` + JenkinsBuildStage_QField_Name + `":{"type":"keyword"},
		"` + JenkinsBuildStage_QField_Success + `":{"type":"boolean"}
	}
}`

//--------------------------------------------------------------------------------

type CFTarget struct {
	Org    string             `json:"org"`
	Space  string             `json:"space"`
	Pushed bool               `json:"pushed"`
	Stage  *JenkinsBuildStage `json:"stage"`
}

const (
	CFTarget_QField_Org    = `org`
	CFTarget_QField_Space  = `space`
	CFTarget_QField_Pushed = `pushed`
	CFTarget_QField_Stage  = `stage`
)

const CFTarget_QMapping = `{
	"type":"nested",
	"dynamic":"strict",
	"properties": {
		"` + CFTarget_QField_Org + `":{"type":"keyword"},
		"` + CFTarget_QField_Space + `":{"type":"keyword"},
		"` + CFTarget_QField_Pushed + `":{"type":"boolean"},
		"` + CFTarget_QField_Stage + `": ` + JenkinsBuildStage_QMapping + `
	}
}`

//--------------------------------------------------------------------------------

type JenkinsBuildTargets struct {
	Id         string     `json:"id"`
	PipelineId string     `json:"pipelineId"`
	Timestamp  string     `json:"timestamp"`
	Build      uint       `json:"build"`
	Sha        string     `json:"sha"`
	Targets    []CFTarget `json:"targets"`
}

const (
	JenkinsBuildTargets_QField_Id         = `id`
	JenkinsBuildTargets_QField_PipelineId = `pipelineId`
	JenkinsBuildTargets_QField_Timestamp  = `timestamp`
	JenkinsBuildTargets_QField_Build      = `build`
	JenkinsBuildTargets_QField_Sha        = `sha`
	JenkinsBuildTargets_QField_CFTargets  = `targets`
)

const JenkinsBuildTargets_QMapping = `{
	"dynamic":"strict",
	"properties":{
		"` + JenkinsBuildTargets_QField_Id + `":{"type":"keyword"},
		"` + JenkinsBuildTargets_QField_PipelineId + `":{"type":"keyword"},
		"` + JenkinsBuildTargets_QField_Timestamp + `":{"type":"keyword"},
		"` + JenkinsBuildTargets_QField_Build + `":{"type":"integer"},
		"` + JenkinsBuildTargets_QField_Sha + `":{"type":"keyword"},
		"` + JenkinsBuildTargets_QField_CFTargets + `": ` + CFTarget_QMapping + `
	}
}`
