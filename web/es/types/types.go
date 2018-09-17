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
	"time"

	c "github.com/venicegeo/vzutil-versioning/common"
)

type Project struct {
	Id          string `json:"id"`
	DisplayName string `json:"displayname"`
}

const ProjectMapping = `{
	"dynamic":"strict",
	"properties":{
		"` + Project_IdField + `":{"type":"keyword"},
		"` + Project_DisplayNameField + `":{"type":"keyword"}
	}
}`
const Project_IdField = `id`
const Project_DisplayNameField = `displayname`

//--------------------------------------------------------------------------------

type Repository struct {
	Id             string                   `json:"id"`
	ProjectId      string                   `json:"project_id"`
	Fullname       string                   `json:"repo"`
	DependencyInfo RepositoryDependencyInfo `json:"depend_info"`
}

type RepositoryDependencyInfo struct {
	RepoFullname string       `json:"repo"`
	CheckoutType CheckoutType `json:"checkout_type"`
	CustomField  string       `json:"custom"`
	FilesToScan  []string     `json:"files"`
}

const RepositoryMapping = `{
	"dynamic":"strict",
	"properties":{
		"` + Repository_IdField + `":{"type":"keyword"},
		"` + Repository_ProjectIdField + `":{"type":"keyword"},
		"` + Repository_NameField + `":{"type":"keyword"},
		"depend_info":{
			"dynamic":"strict",
			"properties":{
				"repo":{"type":"keyword"},
				"checkout_type":{"type":"keyword"},
				"custom":{"type":"keyword"},
				"files":{"type":"keyword"}
			}
		}
	}
}`
const Repository_IdField = `id`
const Repository_ProjectIdField = `project_id`
const Repository_NameField = `repo`

type CheckoutType string

const IncomingSha CheckoutType = "IncomingSha"
const SameRef CheckoutType = "SameRef"
const CustomRef CheckoutType = "CustomRef"
const ExactSha CheckoutType = "ExactSha"

//--------------------------------------------------------------------------------

type Scan struct {
	RepoFullname string            `json:"repo"`
	ProjectId    string            `json:"project_id"`
	Refs         []string          `json:"refs"`
	Sha          string            `json:"sha"`
	Timestamp    time.Time         `json:"timestamp"`
	Scan         *c.DependencyScan `json:"scan"`
}

const Scan_FullnameField = "repo"
const Scan_ProjectIdField = "project_id"
const Scan_RefsField = "refs"
const Scan_ShaField = "sha"
const Scan_TimestampField = "timestamp"
const Scan_SubDependenciesField = "scan." + c.DependenciesField
const Scan_SubFullNameField = "scan." + c.FullNameField
const Scan_SubFilesField = "scan." + c.FilesField

const ScanMapping string = `{
	"dynamic":"strict",
	"properties":{
		"` + Scan_FullnameField + `":{"type":"keyword"},
		"` + Scan_ProjectIdField + `":{"type":"keyword"},
		"` + Scan_RefsField + `":{"type":"keyword"},
		"` + Scan_ShaField + `":{"type":"keyword"},
		"` + Scan_TimestampField + `":{"type":"keyword"},
		"scan":` + c.DependencyScanMapping + `
	}
}`
