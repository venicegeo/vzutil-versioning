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

func Join(strs ...string) string {
	return strings.Join(strs, ".")
}

//--------------------------------------------------------------------------------

type PipelineEntry struct {
	Id           string   `json:"id"`
	Project      string   `json:"project"`       //The project the repo is in
	Repository   string   `json:"repository"`    //The repository org/repo
	PipelineInfo []string `json:"pipeline_info"` //The job parts from the jenkins url
}

const (
	PipelineEntry_IdField           = `id`
	PipelineEntry_ProjectField      = `project`
	PipelineEntry_RepositoryField   = `repository`
	PipelineEntry_PipelineInfoField = `pipeline_info`
)

const PipelineEntryMapping = `{
	"dynamic":"strict",
	"properties":{
		"` + PipelineEntry_IdField + `":{"type":"keyword"},
		"` + PipelineEntry_ProjectField + `":{"type":"keyword"},
		"` + PipelineEntry_RepositoryField + `":{"type":"keyword"},
		"` + PipelineEntry_PipelineInfoField + `":{"type":"keyword"}
	}
}`

//--------------------------------------------------------------------------------

type Stage struct {
	Name    string `json:"name"`
	Success bool   `json:"success"`
}

const (
	Stage_NameField    = `name`
	Stage_SuccessField = `success`
)

const StageMapping = `{
	"dynamic":"strict",
	"properties": {
		"` + Stage_NameField + `":{"type":"keyword"},
		"` + Stage_SuccessField + `":{"type":"boolean"}
	}
}`

//--------------------------------------------------------------------------------

type CFTarget struct {
	Org    string `json:"org"`
	Space  string `json:"space"`
	Pushed bool   `json:"pushed"`
	Stage  *Stage `json:"stage"`
}

const (
	CFTarget_OrgField    = `org`
	CFTarget_SpaceField  = `space`
	CFTarget_PushedField = `pushed`
	CFTarget_StageField  = `stage`
)

const CFTargetMapping = `{
	"type":"nested",
	"dynamic":"strict",
	"properties": {
		"` + CFTarget_OrgField + `":{"type":"keyword"},
		"` + CFTarget_SpaceField + `":{"type":"keyword"},
		"` + CFTarget_PushedField + `":{"type":"boolean"},
		"` + CFTarget_StageField + `": ` + StageMapping + `
	}
}`

//--------------------------------------------------------------------------------

type Targets struct {
	Id        string     `json:"id"`
	RepoId    string     `json:"repoId"`
	Timestamp string     `json:"timestamp"`
	Build     uint       `json:"build"`
	Sha       string     `json:"sha"`
	Targets   []CFTarget `json:"targets"`
}

const (
	Targets_IdField        = `id`
	Targets_RepoIdField    = `repoId`
	Targets_TimestampField = `timestamp`
	Targets_BuildField     = `build`
	Targets_ShaField       = `sha`
	Targets_CFTargets      = `targets`
)

const TargetsMapping = `{
	"dynamic":"strict",
	"properties":{
		"` + Targets_IdField + `":{"type":"keyword"},
		"` + Targets_RepoIdField + `":{"type":"keyword"},
		"` + Targets_TimestampField + `":{"type":"keyword"},
		"` + Targets_BuildField + `":{"type":"integer"},
		"` + Targets_ShaField + `":{"type":"keyword"},
		"` + Targets_CFTargets + `": ` + CFTargetMapping + `
	}
}`
