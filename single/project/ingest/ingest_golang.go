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
	"github.com/venicegeo/vzutil-versioning/single/project/dependency"
	"github.com/venicegeo/vzutil-versioning/single/project/issue"
	lan "github.com/venicegeo/vzutil-versioning/single/project/language"
)

type GoProjectWrapper struct {
	Yaml *GlideYaml
	Lock *GlideLock
	ProjectWrapper
}

func (pw *GoProjectWrapper) compileCheck() {
	var _ IProjectWrapper = (*GoProjectWrapper)(nil)
}

func (pw *GoProjectWrapper) GetResults() (dependency.GenericDependencies, []*issue.Issue, error) {
	yamlArray := append(pw.Yaml.Dependences, pw.Yaml.TestDependences...)
	lockArray := append(pw.Lock.Packages, pw.Lock.TestPackages...)

	var deps dependency.GenericDependencies
	version := ""
	for _, elem := range yamlArray {
		version = elem.Version
		for _, lock := range lockArray {
			if elem.Name == lock.Name && elem.Version != lock.Sha {
				pw.addIssue(issue.NewVersionMismatch(elem.Name, version, lock.Sha))
				version = lock.Sha
				break
			}
		}
		deps.Add(dependency.NewGenericDependency(elem.Name, version, pw.name, lan.Go))
	}
	return deps, pw.issues, nil
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
