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
	Id          string `json:"id"`
	DisplayName string `json:"displayname"`
}

const ProjectMapping = `{
	"dynamic":"strict",
	"properties":{
		"` + ProjectIdField + `":{"type":"keyword"},
		"` + ProjectDisplayNameField + `":{"type":"keyword"}
	}
}`
const ProjectIdField = `id`
const ProjectDisplayNameField = `displayname`

//-----------------------------------------------------------------

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
		"id":{"type":"keyword"},
		"project_id":{"type":"keyword"},
		"repo":{"type":"keyword"},
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
const RepositoryIdField = `id`
const RepositoryProjectIdField = `project_id`
const RepositoryNameField = `repo`

type CheckoutType string

const IncomingSha CheckoutType = "IncomingSha"
const SameRef CheckoutType = "SameRef"
const CustomRef CheckoutType = "CustomRef"
const ExactSha CheckoutType = "ExactSha"
