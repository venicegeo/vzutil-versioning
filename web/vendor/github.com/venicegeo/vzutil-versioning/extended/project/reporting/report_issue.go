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
package report

import (
	"sort"

	"github.com/venicegeo/vzutil-versioning/plural/project"
	"github.com/venicegeo/vzutil-versioning/plural/project/issue"
	"gopkg.in/yaml.v2"
)

const YmlIssuesHeader = "#\n# Issues generated from current piazza versions\n#\n"

type YmlIssuesWrapper struct {
	issue.IssuesMap `yaml:"issues"`
}

func GenerateIssuesYaml(projects *project.Projects) ([]byte, error) {
	issuesMap := issue.IssuesMap{}
	for k, v := range *projects {
		if len(v.Issues) == 0 {
			continue
		}
		issues := issue.Issues{}
		for _, issue := range v.Issues {
			issues = append(issues, issue.String())
		}
		issuesMap[k] = &issues
	}
	for _, v := range issuesMap {
		sort.Strings(*v)
	}
	return yaml.Marshal(YmlIssuesWrapper{issuesMap})
}
