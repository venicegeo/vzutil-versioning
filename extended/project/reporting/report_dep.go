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

	deps "github.com/venicegeo/vzutil-versioning/plural/project/dependency"

	"gopkg.in/yaml.v2"
)

const YmlDependenciesHeader = "#\n# All versions of dependencies per project\n#\n"

type YmlDependenciesWrapper struct {
	YmlMap `yaml:"dependencies"`
}

type YmlMap map[string][]string

func GenerateDependenciesYaml(depens *deps.GenericDependencies) ([]byte, error) {
	dependenciesMap := YmlMap{}
	for _, dep := range *depens {
		if _, ok := dependenciesMap[dep.GetProject()]; !ok {
			dependenciesMap[dep.GetProject()] = []string{}
		}
		dependenciesMap[dep.GetProject()] = append(dependenciesMap[dep.GetProject()], dep.FullString()) //dep.String())
	}
	for _, v := range dependenciesMap {
		sort.Strings(v)
	}
	return yaml.Marshal(YmlDependenciesWrapper{dependenciesMap})
}
