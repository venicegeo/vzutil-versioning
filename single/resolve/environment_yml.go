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
	"errors"
	"regexp"
	"sort"

	d "github.com/venicegeo/vzutil-versioning/common/dependency"
	i "github.com/venicegeo/vzutil-versioning/common/issue"
	lan "github.com/venicegeo/vzutil-versioning/common/language"
	"gopkg.in/yaml.v2"
)

var environment_splitRE = regexp.MustCompile(`^([^>=<]+)((?:(?:<=)|(?:>=))|(?:=))?(.+)?$`)

func (r *Resolver) ResolveEnvironmentYml(location string, test bool) (d.Dependencies, i.Issues, error) {
	dat, err := r.readFile(location)
	if err != nil {
		return nil, nil, err
	}
	env := CondaEnvironment{}
	if err := yaml.Unmarshal(dat, &env); err != nil {
		return nil, nil, err
	}
	condaLines, pipLines, err := env.getDepLines()
	if err != nil {
		return nil, nil, err
	}
	deps := make(d.Dependencies, 0, len(condaLines)+len(pipLines))
	issues := i.Issues{}
	for _, dep := range condaLines {
		dep, _ := r.parseCondaLine(dep, &issues)
		deps = append(deps, dep)
	}
	for _, dep := range pipLines {
		if dep, ok := r.parsePipLine(dep, &issues); ok {
			deps = append(deps, dep)
		}
	}
	sort.Sort(deps)
	sort.Sort(issues)
	return deps, issues, nil
}

type CondaEnvironment struct {
	Name         string        `yaml:"name"`
	Channels     []string      `yaml:"channels"`
	Dependencies []interface{} `yaml:"dependencies"`
}

func (c *CondaEnvironment) getDepLines() ([]string, []string, error) {
	condaLines := []string{}
	pipLines := []string{}
	for _, d := range c.Dependencies {
		switch d.(type) {
		case string:
			condaLines = append(condaLines, d.(string))
		case map[interface{}]interface{}:
			pip, ok := d.(map[interface{}]interface{})["pip"]
			if !ok {
				return nil, nil, errors.New("Map found in yml not containing pip key")
			}
			pipDeps, ok := pip.([]interface{})
			if !ok {
				return nil, nil, errors.New("Pip entry not []interface{}")
			}
			for _, dep := range pipDeps {
				if str, ok := dep.(string); !ok {
					return nil, nil, errors.New("Pip dependency non type string")
				} else {
					pipLines = append(pipLines, str)
				}
			}
		default:
			return nil, nil, errors.New("Unknown type found in yml")
		}
	}
	return condaLines, pipLines, nil
}

func (r *Resolver) parseCondaLine(line string, issues *i.Issues) (d.Dependency, bool) {
	parts := environment_splitRE.FindStringSubmatch(line)[1:]
	if parts[1] != "=" {
		*issues = append(*issues, i.NewWeakVersion(parts[0], parts[2], parts[1]))
	}
	return d.NewDependency(parts[0], parts[2], lan.Conda), true
}
