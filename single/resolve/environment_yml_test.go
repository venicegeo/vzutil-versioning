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
	"reflect"
	"testing"

	d "github.com/venicegeo/vzutil-versioning/common/dependency"
	i "github.com/venicegeo/vzutil-versioning/common/issue"
	l "github.com/venicegeo/vzutil-versioning/common/language"
)

var environmentYmlTestData = map[string]string{}
var environmentYmlTestResults = map[string]ResolveResult{}

func readEnvironmentYml(file string) ([]byte, error) {
	return []byte(environmentYmlTestData[file]), nil
}

func TestEnvironmentYml(t *testing.T) {
	resolver := NewResolver(readEnvironmentYml)
	for k, _ := range environmentYmlTestData {
		expected := environmentYmlTestResults[k]
		d, i, e := resolver.ResolveEnvironmentYml(k, false)
		if !reflect.DeepEqual(e, expected.err) {
			t.Fatal(e, "not equal to", expected.err)
		} else if !reflect.DeepEqual(d, expected.deps) {
			t.Fatal(d, "not equal to", expected.deps)
		} else if !reflect.DeepEqual(i, expected.issues) {
			t.Fatal(i, "not equal to", expected.issues)
		}
	}
}

func setupEnvironmentYml1() {
	environmentYmlTestData["1"] = `
name: test_one
channels:
    - conda-forge
dependencies:
  - click=6.6
  - numpy=1.14.0=py27_blas_openblas_200
  - pytides
`
	environmentYmlTestResults["1"] = ResolveResult{
		deps:   d.Dependencies{d.NewDependency("click", "6.6", l.Conda), d.NewDependency("numpy", "1.14.0=py27_blas_openblas_200", l.Conda), d.NewDependency("pytides", "", l.Conda)},
		issues: i.Issues{i.NewWeakVersion("pytides", "", "")},
		err:    nil,
	}
}

func setupEnvironmentYml2() {
	environmentYmlTestData["2"] = `
name: test_two
dependencies:
    - gdal=2.1.3
    - pip=1.2
    - pip=1.3
    - setuptools=0
`
	environmentYmlTestResults["2"] = ResolveResult{
		deps:   d.Dependencies{d.NewDependency("gdal", "2.1.3", l.Conda), d.NewDependency("pip", "1.2", l.Conda), d.NewDependency("pip", "1.3", l.Conda), d.NewDependency("setuptools", "0", l.Conda)},
		issues: i.Issues{},
		err:    nil,
	}
}

func setupEnvironmentYml3() {
	environmentYmlTestData["3"] = `
name: test_three
dependencies:
    - gippy=1.0.0.post3
    - pip=1.0
    - bfalg-ndwi=2.0.0
    - pip:
      - setuptools==39.0.0
      - git+https://github.com/happy/place.git@v1.0.1#egg=some-thing
`
	environmentYmlTestResults["3"] = ResolveResult{
		deps:   d.Dependencies{d.NewDependency("gippy", "1.0.0.post3", l.Conda), d.NewDependency("pip", "1.0", l.Conda), d.NewDependency("bfalg-ndwi", "2.0.0", l.Conda), d.NewDependency("setuptools", "29.0.0", l.Python)},
		issues: i.Issues{},
		err:    nil,
	}
}
