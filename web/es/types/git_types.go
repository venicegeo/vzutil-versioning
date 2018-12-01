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
	"regexp"
	"time"

	c "github.com/venicegeo/vzutil-versioning/common"
)

var escape = regexp.MustCompile(`[^a-zA-Z\-_]`)

type Project struct {
	Id          string `json:"id"`
	DisplayName string `json:"displayname"`
	EscapedName string `json:"escapedname"`
}

const Project_QMapping = `{
	"dynamic":"strict",
	"properties":{
		"` + Project_QField_Id + `":{"type":"keyword"},
		"` + Project_QField_DisplayName + `":{"type":"keyword"},
		"` + Project_QField_EscapedName + `":{"type":"keyword"}
	}
}`
const (
	Project_QField_Id          = `id`
	Project_QField_DisplayName = `displayname`
	Project_QField_EscapedName = `escapedname`
)

func NewProject(id, name string) Project {
	return Project{id, name, escape.ReplaceAllString(name, "_")}
}

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

const Repository_QMapping = `{
	"dynamic":"strict",
	"properties":{
		"` + Repository_QField_Id + `":{"type":"keyword"},
		"` + Repository_QField_ProjectId + `":{"type":"keyword"},
		"` + Repository_QField_Name + `":{"type":"keyword"},
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
const (
	Repository_QField_Id        = `id`
	Repository_QField_ProjectId = `project_id`
	Repository_QField_Name      = `repo`
)

type CheckoutType string

const (
	IncomingSha CheckoutType = "IncomingSha"
	SameRef     CheckoutType = "SameRef"
	CustomRef   CheckoutType = "CustomRef"
	ExactSha    CheckoutType = "ExactSha"
)

//--------------------------------------------------------------------------------

type Scan struct {
	RepoFullname string            `json:"repo"`
	ProjectId    string            `json:"project_id"`
	Refs         []string          `json:"refs"`
	Sha          string            `json:"sha"`
	Timestamp    time.Time         `json:"timestamp"`
	Scan         *c.DependencyScan `json:"scan"`
}

const (
	Scan_QField_Fullname        = "repo"
	Scan_QField_ProjectId       = "project_id"
	Scan_QField_Refs            = "refs"
	Scan_QField_Sha             = "sha"
	Scan_QField_Timestamp       = "timestamp"
	Scan_QField_SubDependencies = "scan." + c.DependenciesField
	Scan_QField_SubFullName     = "scan." + c.FullNameField
	Scan_QField_SubFiles        = "scan." + c.FilesField
)

const Scan_QMapping string = `{
	"dynamic":"strict",
	"properties":{
		"` + Scan_QField_Fullname + `":{"type":"keyword"},
		"` + Scan_QField_ProjectId + `":{"type":"keyword"},
		"` + Scan_QField_Refs + `":{"type":"keyword"},
		"` + Scan_QField_Sha + `":{"type":"keyword"},
		"` + Scan_QField_Timestamp + `":{"type":"keyword"},
		"scan":` + c.DependencyScanMapping + `
	}
}`
