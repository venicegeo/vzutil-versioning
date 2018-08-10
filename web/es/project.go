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

package es

type Project struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayname"`
}

type ProjectEntry struct {
	ProjectName        string   `json:"project_name"`
	RepoFullname       string   `json:"repo"`
	DependRepoFullname string   `json:"depend_repo"`
	FilesToScan        []string `json:"files"`
}

const ProjectEntryMapping = `{
	"dynamic":"strict",
	"properties":{
		"project_name":{"type":"keyword"},
		"repo":{"type":"keyword"},
		"depend_repo":{"type":"keyword"},
		"files":{"type":"keyword"}
	}
}`
const ProjectMapping = `{
	"dynamic":"strict",
	"properties":{
		"name":{"type":"keyword"},
		"displayname":{"type":"keyword"}
	}
}`
