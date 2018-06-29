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

func ResolveGlideYaml(location string, test bool) ([]*dependency.GenericDependency, []*issue.Issue, error) {
	yamlDat, err := ioutil.ReadFile(location)
	if err != nil {
		return nil, nil, err
	}
	lockDat, err := ioutil.ReadFile(strings.TrimSuffix(location, ".yaml") + ".lock")
	if err != nil {
		return nil, nil, err
	}
	var yml GlideYaml
	if err := yaml.Unmarshal(yamlDat, &yml); err != nil {
		return nil, nil, err
	}
	var lock GlideLock
	if err := yaml.Unmarshal(lockDat, &lock); err != nil {
		return nil, nil, err
	}

	yamlArray := yml.Dependences
	lockArray := lock.Packages
	if test {
		yamlArray = append(yamlArray, yml.TestDependences...)
		lockArray = append(lockArray, lock.TestPackages...)
	}

	deps := make([]*dependency.GenericDependency, len(yamlArray), len(yamlArray))
	issues := []*issue.Issue{}
	var version string
	for i, elem := range yamlArray {
		version = elem.Version
		if version == "" {
			issues = append(issues, issue.NewMissingVersion(elem.Name))
			for _, lock := range lockArray {
				if elem.Name == lock.Name {
					version = lock.Sha
					break
				}
			}
		}
		deps[i] = dependency.NewGenericDependency(elem.Name, version, lan.Go)
	}
	return deps, issues, nil
}

type GlideYaml struct {
	BasePackage     string            `yaml:"package"`
	Dependences     []GlideDependence `yaml:"import"`
	TestDependences []GlideDependence `yaml:"testImport"`
}

//----------------------------------------------------------------------------

type GlideDependences map[string]GlideDependence

type GlideDependence struct {
	Name    string `yaml:"package"`
	Version string `yaml:"version"`
}

//----------------------------------------------------------------------------

type GlideLock struct {
	Hash         string
	Updated      string
	Packages     []GlidePackage `yaml:"imports"`
	TestPackages []GlidePackage `yaml:"testImports"`
}

type GlidePackage struct {
	Name        string
	Path        string
	Sha         string `yaml:"version"`
	Subpackages []string
}

//----------------------------------------------------------------------------
