/*
Copyright 2017, RadiantBlue Technologies, Inc.

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
package ingest

import (
	"github.com/venicegeo/vzutil-versioning/common/dependency"
	"github.com/venicegeo/vzutil-versioning/single/project/issue"
)

type ProjectWrapper struct {
	issues   []*issue.Issue
	name     string
	location string
}

func (pw *ProjectWrapper) SetProperties(location, name string) {
	pw.location = location
	pw.name = name
}
func (pw *ProjectWrapper) addIssue(iss *issue.Issue) {
	pw.issues = append(pw.issues, iss)
}

type IProjectWrapper interface {
	compileCheck()
	SetProperties(string, string)
	GetResults() (dependency.GenericDependencies, []*issue.Issue, error)
	addIssue(*issue.Issue)
}
