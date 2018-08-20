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
	"regexp"
	"sort"
	"strings"

	d "github.com/venicegeo/vzutil-versioning/common/dependency"
	i "github.com/venicegeo/vzutil-versioning/common/issue"
	lan "github.com/venicegeo/vzutil-versioning/common/language"
	"github.com/venicegeo/vzutil-versioning/single/util"
)

var requirements_gitRE = regexp.MustCompile(`^git(?:(?:\+https)|(?:\+ssh)|(?:\+git))*:\/\/(?:git\.)*github\.com\/.+\/([^@.]+)()(?:(?:.git)?@([^#]+))?`)
var requirements_elseRE = regexp.MustCompile(`^([^>=<]+)((?:(?:<=)|(?:>=))|(?:==))?(.+)?$`)

func (r *Resolver) ResolveRequirementsTxt(location string, test bool) (d.Dependencies, i.Issues, error) {
	dat, err := r.readFile(location)
	if err != nil {
		return nil, nil, err
	}
	lines := util.StringSliceTrimSpaceRemoveEmpty(strings.Split(string(dat), "\n"))
	deps := make(d.Dependencies, 0, len(lines))
	issues := i.Issues{}
	for _, line := range lines {
		if dep, ok := r.parsePipLine(line, &issues); ok {
			deps = append(deps, dep)
		}
	}
	sort.Sort(deps)
	sort.Sort(issues)
	return deps, issues, nil
}

func (r *Resolver) parsePipLine(line string, issues *i.Issues) (d.Dependency, bool) {
	if line == "" || strings.Contains(line, "lib/python") || strings.HasPrefix(line, "-r") || strings.HasPrefix(line, "#") {
		return d.Dependency{}, false
	}
	parts := []string{}
	if requirements_gitRE.MatchString(line) {
		parts = requirements_gitRE.FindStringSubmatch(line)[1:]
	} else {
		parts = requirements_elseRE.FindStringSubmatch(line)[1:]
		if parts[1] != "==" {
			*issues = append(*issues, i.NewWeakVersion(parts[0], parts[2], parts[1]))
		}
	}
	return d.NewDependency(parts[0], parts[2], lan.Python), true
}
