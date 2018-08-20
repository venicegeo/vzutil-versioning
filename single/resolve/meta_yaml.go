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
	"sort"
	"strings"

	"regexp"

	d "github.com/venicegeo/vzutil-versioning/common/dependency"
	i "github.com/venicegeo/vzutil-versioning/common/issue"
	lan "github.com/venicegeo/vzutil-versioning/common/language"
	"github.com/venicegeo/vzutil-versioning/single/util"
	"gopkg.in/yaml.v2"
)

var findRecipeVarDeclaration = regexp.MustCompile(`{% set ([^ ]+) = "([^"]+)" %}`)

func (r *Resolver) ResolveMetaYaml(location string, test bool) (d.Dependencies, i.Issues, error) {
	dat, err := r.readFile(location)
	if err != nil {
		return nil, nil, err
	}
	str := string(dat)
	variables := findRecipeVarDeclaration.FindAllStringSubmatch(str, -1)
	for _, varr := range variables {
		str = strings.Replace(str, varr[0], "", -1)
		str = strings.Replace(str, "{{ "+varr[1]+" }}", varr[2], -1)
	}
	dat = []byte(str)
	var recipe CondaRecipe
	if err := yaml.Unmarshal(dat, &recipe); err != nil {
		return nil, nil, err
	}
	deps := make(d.Dependencies, 0, len(recipe.Requirements.Build)+len(recipe.Requirements.Run)+len(recipe.Requirements.Host))
	issues := i.Issues{}
	for _, s := range append(append(recipe.Requirements.Build, recipe.Requirements.Run...), recipe.Requirements.Host...) {
		parts := util.SplitAtAnyTrim(s, " ", "=")
		if len(parts) == 1 {
			parts = append(parts, "")
			issues = append(issues, i.NewMissingVersion(parts[0]))
		}
		deps = append(deps, d.NewDependency(parts[0], strings.Join(parts[1:], "="), lan.Conda))
	}
	d.RemoveExactDuplicates(&deps)
	sort.Sort(deps)
	sort.Sort(issues)
	return deps, issues, nil
}

type CondaRecipe struct {
	Requirements struct {
		Build []string `json:"build"`
		Run   []string `json:"run"`
		Host  []string `json:"host"`
	} `json:"requirements"`
}
