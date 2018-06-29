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
	"io/ioutil"
	"regexp"
	"strings"

	"github.com/venicegeo/vzutil-versioning/common/dependency"
	"github.com/venicegeo/vzutil-versioning/common/issue"
	lan "github.com/venicegeo/vzutil-versioning/common/language"
	"github.com/venicegeo/vzutil-versioning/single/util"
)

var requirements_gitRE = regexp.MustCompile(`^git(?:(?:\+https)|(?:\+ssh)|(?:\+git))*:\/\/(?:git\.)*github\.com\/.+\/([^@.]+)()(?:(?:.git)?@([^#]+))?`)
var requirements_elseRE = regexp.MustCompile(`^([^>=<]+)((?:(?:<=)|(?:>=))|(?:==))?(.+)?$`)

func ResolveRequirementsTxt(location string, test bool) ([]*dependency.GenericDependency, []*issue.Issue, error) {
	dat, err := ioutil.ReadFile(location)
	if err != nil {
		return nil, nil, err
	}
	lines := util.StringSliceTrimSpaceRemoveEmpty(strings.Split(string(dat), "\n"))
	deps := make([]*dependency.GenericDependency, len(lines), len(lines))
	issues := []*issue.Issue{}
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.Contains(line, "lib/python") || strings.HasPrefix(line, "-r") || strings.HasPrefix(line, "#") {
			continue
		}
		parts := []string{}
		if requirements_gitRE.MatchString(line) {
			parts = requirements_gitRE.FindStringSubmatch(line)[1:]
		} else {
			parts = requirements_elseRE.FindStringSubmatch(line)[1:]
			if parts[1] != "==" {
				issues = append(issues, issue.NewWeakVersion(parts[0], parts[2], parts[1]))
			}
		}
		deps[i] = dependency.NewGenericDependency(parts[0], parts[2], lan.Python)
	}
	return deps, issues, nil
}
