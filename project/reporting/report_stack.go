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

	deps "github.com/venicegeo/vzutil-versioning/project/dependency"

	"gopkg.in/yaml.v2"
)

const YmlAboutHeader = "#\n# Mock yml file generated from current piazza versions\n#\n"

type YmlStacksWrapper struct {
	YmlMap `yaml:"stacks"`
}

func GenerateStacksYaml(depens *deps.GenericDependencies) ([]byte, error) {
	dependenciesMap := YmlMap{}
	for _, dep := range *depens {
		if _, ok := dependenciesMap[string(dep.GetLanguage())]; !ok {
			dependenciesMap[string(dep.GetLanguage())] = []string{}
		}
		dependenciesMap[string(dep.GetLanguage())] = append(dependenciesMap[string(dep.GetLanguage())], dep.String())
	}
	for _, v := range dependenciesMap {
		sort.Strings(v)
	}
	return yaml.Marshal(YmlStacksWrapper{dependenciesMap})
}
