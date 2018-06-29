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
package resolve

import (
	"encoding/json"
	"io/ioutil"
	"regexp"
	"strings"

	"github.com/venicegeo/vzutil-versioning/common/dependency"
	"github.com/venicegeo/vzutil-versioning/common/issue"
	lan "github.com/venicegeo/vzutil-versioning/common/language"
)

var package_gitRE = regexp.MustCompile(`^git(?:(?:\+(?:https)|(?:ssh))|(?:\+ssh))*:\/\/(?:git\.)*github\.com\/.+\/.+\.git(?:#(.+))?`)
var package_elseRE = regexp.MustCompile(`^((?:>=)|(?:<=)|(?:>)|(?:<)|(?:~)|(?:\^))*.+$`)

type PackageJson struct {
	DependencyMap    map[string]string `json:"dependencies"`
	DevDependencyMap map[string]string `json:"devDependencies"`
}

func ResolvePackageJson(location string, test bool) ([]*dependency.GenericDependency, []*issue.Issue, error) {
	dat, err := ioutil.ReadFile(location)
	if err != nil {
		return nil, nil, err
	}
	var packageJson PackageJson
	if err := json.Unmarshal(dat, &packageJson); err != nil {
		return nil, nil, err
	}
	depMap := packageJson.DependencyMap
	if test {
		for k, v := range packageJson.DevDependencyMap {
			depMap[k] = v
		}
	}
	deps := make([]*dependency.GenericDependency, 0, len(depMap))
	issues := []*issue.Issue{}
	for name, version := range depMap {
		if package_gitRE.MatchString(version) {
			version = package_gitRE.FindStringSubmatch(version)[1]
		} else {
			tag := package_elseRE.FindStringSubmatch(version)[1]
			if tag != "" {
				issues = append(issues, issue.NewWeakVersion(name, version, tag))
				version = strings.TrimPrefix(version, tag)
			}
		}
		deps = append(deps, dependency.NewGenericDependency(name, version, lan.JavaScript))
	}
	return deps, issues, nil
}
