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
	"strings"

	"github.com/venicegeo/vzutil-versioning/common/dependency"
	"github.com/venicegeo/vzutil-versioning/common/issue"
	lan "github.com/venicegeo/vzutil-versioning/common/language"
	"gopkg.in/yaml.v2"
)

func ResolveMetaYaml(location string, test bool) ([]*dependency.GenericDependency, []*issue.Issue, error) {
	dat, err := ioutil.ReadFile(location)
	if err != nil {
		return nil, nil, err
	}
	var recipe CondaRecipe
	if err := yaml.Unmarshal(dat, &recipe); err != nil {
		return nil, nil, err
	}
	comb := make([][]string, 0, len(recipe.Requirements.Build)+len(recipe.Requirements.Run))
	for _, s := range append(recipe.Requirements.Build, recipe.Requirements.Run...) {
		parts := strings.Split(s, " ")
		if len(parts) == 1 {
			parts = append(parts, "")
		}
		comb = append(comb, parts)
	}
	unique := [][2]string{}
	duplicate := false
	for _, p := range comb {
		for _, c := range unique {
			if p[0] == c[0] && c[1] == c[1] {
				duplicate = true
				break
			}
		}
		if !duplicate {
			unique = append(unique, [2]string{p[0], p[1]})
		}
	}
	deps := make([]*dependency.GenericDependency, len(unique), len(unique))
	issues := []*issue.Issue{}
	for i, u := range unique {
		deps[i] = dependency.NewGenericDependency(u[0], u[1], lan.Python)
		if u[1] == "" {
			issues = append(issues, issue.NewMissingVersion(u[0]))
		}
	}
	return deps, issues, nil
}

type CondaRecipe struct {
	Requirements struct {
		Build []string `json:"build"`
		Run   []string `json:"run"`
	} `json:"requirements"`
}
