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
	"gopkg.in/yaml.v2"
)

type PipProjectWrapper struct {
	Filedat    []byte
	DevFileDat []byte
	ProjectWrapper
}
type CondaProjectWrapper struct {
	Filedat []byte
	ProjectWrapper
}

func (pw *PipProjectWrapper) compileCheck() {
	var _ IProjectWrapper = (*PipProjectWrapper)(nil)
}
func (pw *CondaProjectWrapper) compileCheck() {
	var _ IProjectWrapper = (*CondaProjectWrapper)(nil)
}

func (pw *PipProjectWrapper) GetResults() (dependency.GenericDependencies, []*issue.Issue, error) {
	deps := dependency.GenericDependencies{}
	gitRE := regexp.MustCompile(`^git(?:(?:\+https)|(?:\+ssh)|(?:\+git))*:\/\/(?:git\.)*github\.com\/.+\/([^@.]+)()(?:(?:.git)?@([^#]+))?`)
	elseRE := regexp.MustCompile(`^([^>=<]+)((?:(?:<=)|(?:>=))|(?:==))?(.+)?$`)
	get := func(str string) {
		for _, line := range strings.Split(str, "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.Contains(line, "lib/python") || strings.HasPrefix(line, "-r") || strings.HasPrefix(line, "#") {
				continue
			}
			parts := []string{}
			if gitRE.MatchString(line) {
				parts = gitRE.FindStringSubmatch(line)[1:]
			} else {
				parts = elseRE.FindStringSubmatch(line)[1:]
				if parts[1] != "==" {
					pw.addIssue(issue.NewWeakVersion(parts[0], parts[2], parts[1]))
				}
			}
			//parts = append(parts, []string{"Unknown", "Unknown"}...)
			//			if len(parts) < 3 {
			//				parts = append(parts, "Unknown")
			//			}
			deps.Add(dependency.NewGenericDependency(parts[0], parts[2], pw.name, lan.Python))
		}
	}
	get(string(pw.Filedat))
	get(string(pw.DevFileDat))
	return deps, pw.issues, nil
}

func (pw *CondaProjectWrapper) GetResults() (dependency.GenericDependencies, []*issue.Issue, error) {
	env := CondaEnvironment{}
	if err := yaml.Unmarshal(pw.Filedat, &env); err != nil {
		return nil, nil, err
	}
	deps := dependency.GenericDependencies{}
	splitRE := regexp.MustCompile(`^([^>=<]+)((?:(?:<=)|(?:>=))|(?:=))?(.+)?$`)
	for _, dep := range env.Dependencies {
		parts := splitRE.FindStringSubmatch(dep)[1:]
		if parts[1] != "=" {
			pw.addIssue(issue.NewWeakVersion(parts[0], parts[2], parts[1]))
		}
		deps.Add(dependency.NewGenericDependency(parts[0], parts[2], pw.name, lan.Python))
	}
	return deps, pw.issues, nil
}

type CondaEnvironment struct {
	Name         string   `yaml:"name"`
	Channels     []string `yaml:"channels"`
	Dependencies []string `yaml:"dependencies"`
}
