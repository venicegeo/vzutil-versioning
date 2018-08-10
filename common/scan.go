/*
Copyright 2018, RadiantBlue Technologies, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package com

import (
	"time"

	d "github.com/venicegeo/vzutil-versioning/common/dependency"
)

const FullNameField = `full_name`
const NameField = `name`
const RefsField = `refs`
const ShaField = `sha`
const TimestampField = `timestamp`
const DependenciesField = `dependencies`
const IssuesField = `issues`
const FilesField = `files`

const DependencyScanMapping string = `{
	"dynamic":"strict",
	"properties":{
		"full_name":{"type":"keyword"},
		"name":{"type":"keyword"},
		"refs":{"type":"keyword"},
		"sha":{"type":"keyword"},
		"timestamp":{"type":"keyword"},
		"dependencies":` + d.DependencyMapping + `,
		"issues":{"type":"keyword"},
		"files":{"type":"keyword"},
		"requester":{"type":"keyword"}
	}
}`

type DependencyScan struct {
	Fullname      string         `json:"full_name"`
	Name          string         `json:"name"`
	Sha           string         `json:"sha"`
	Refs          []string       `json:"refs"`
	Deps          []d.Dependency `json:"dependencies"`
	Issues        []string       `json:"issues"`
	Files         []string       `json:"files"`
	Timestamp     time.Time      `json:"timestamp"`
	RequesterName string         `json:"requester"`
}

type DependencyScans map[string]DependencyScan
