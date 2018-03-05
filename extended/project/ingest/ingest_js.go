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
	"regexp"
	"strings"

	"github.com/venicegeo/vzutil-versioning/plural/project/dependency"
	"github.com/venicegeo/vzutil-versioning/plural/project/issue"
	lan "github.com/venicegeo/vzutil-versioning/plural/project/language"
)

type JsProjectWrapper struct {
	DependencyMap    map[string]string `json:"dependencies"`
	DevDependencyMap map[string]string `json:"devDependencies"`
	ProjectWrapper
}

func (pw *JsProjectWrapper) compileCheck() {
	var _ IProjectWrapper = (*JsProjectWrapper)(nil)
}

func (pw *JsProjectWrapper) GetResults() (deps dependency.GenericDependencies, issues []*issue.Issue, err error) {
	gitRE := regexp.MustCompile(`^git(?:(?:\+(?:https)|(?:ssh))|(?:\+ssh))*:\/\/(?:git\.)*github\.com\/.+\/.+\.git(?:#(.+))?`)
	elseRE := regexp.MustCompile(`^((?:>=)|(?:<=)|(?:>)|(?:<)|(?:~)|(?:\^))*.+$`)
	for _, mp := range []map[string]string{pw.DependencyMap, pw.DevDependencyMap} {
		for name, version := range mp {
			if gitRE.MatchString(version) {
				version = gitRE.FindStringSubmatch(version)[1]
			} else {
				tag := elseRE.FindStringSubmatch(version)[1]
				if tag != "" {
					pw.addIssue(issue.NewWeakVersion(name, version, tag))
					version = strings.TrimPrefix(version, tag)
				}
			}
			deps.Add(dependency.NewGenericDependency(name, version, pw.name, lan.JavaScript))
		}
	}
	return deps, issues, nil
}
