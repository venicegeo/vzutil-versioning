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
package ingest

import (
	"errors"
	"io/ioutil"
	"regexp"
	"strings"

	"github.com/venicegeo/vzutil-versioning/common/dependency"
	lan "github.com/venicegeo/vzutil-versioning/common/language"
	"github.com/venicegeo/vzutil-versioning/single/project/issue"
	"gopkg.in/yaml.v2"
)

var environment_splitRE = regexp.MustCompile(`^([^>=<]+)((?:(?:<=)|(?:>=))|(?:=))?(.+)?$`)

func ResolveEnvironmentYml(location string, test bool) ([]*dependency.GenericDependency, []*issue.Issue, error) {
	dat, err := ioutil.ReadFile(location)
	if err != nil {
		return nil, nil, err
	}
	envw := CondaEnvironmentWrapper{}
	if err := yaml.Unmarshal(dat, &envw); err != nil {
		return nil, nil, err
	}
	env, err := envw.convert()
	if err != nil {
		return nil, nil, err
	}
	deps := make([]*dependency.GenericDependency, len(env.Dependencies), len(env.Dependencies))
	issues := []*issue.Issue{}
	for i, dep := range env.Dependencies {
		parts := environment_splitRE.FindStringSubmatch(dep)[1:]
		if parts[1] != "=" {
			issues = append(issues, issue.NewWeakVersion(parts[0], parts[2], parts[1]))
		}
		deps[i] = dependency.NewGenericDependency(parts[0], strings.Split(parts[2], "=")[0], lan.Python)
	}
	return deps, issues, nil
}

type CondaEnvironmentWrapper struct {
	Name         string        `yaml:"name"`
	Channels     []string      `yaml:"channels"`
	Dependencies []interface{} `yaml:"dependencies"`
}
type CondaEnvironment struct {
	Name         string   `yaml:"name"`
	Channels     []string `yaml:"channels"`
	Dependencies []string `yaml:"dependencies"`
}

func (c *CondaEnvironmentWrapper) convert() (*CondaEnvironment, error) {
	ret := &CondaEnvironment{c.Name, c.Channels, []string{}}
	for _, d := range c.Dependencies {
		switch d.(type) {
		case string:
			ret.Dependencies = append(ret.Dependencies, d.(string))
		case map[interface{}]interface{}:
			pip, ok := d.(map[interface{}]interface{})["pip"]
			if !ok {
				return nil, errors.New("Map found in yml not containing pip key")
			}
			pipDeps, ok := pip.([]interface{})
			if !ok {
				return nil, errors.New("Pip entry not []interface{}")
			}
			for _, dep := range pipDeps {
				if str, ok := dep.(string); !ok {
					return nil, errors.New("Pip dependency non type string")
				} else {
					ret.Dependencies = append(ret.Dependencies, str)
				}
			}
		default:
			return nil, errors.New("Unknown type found in yml")
		}
	}
	return ret, nil
}
