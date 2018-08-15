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

var requirementsTxtTestData = map[string]string{}
var requirementsTxtTestResults = map[string]ResolveResult{}

func readRequirementsTxt(file string) ([]byte, error) {
	return []byte(requirementsTxtTestData[file]), nil
}

func TestRequirementsTxt(t *testing.T) {
	resolver := NewResolver(readRequirementsTxt)
	for k, _ := range requirementsTxtTestData {
		expected := requirementsTxtTestResults[k]
		d, i, e := resolver.ResolveRequirementsTxt(k, false)
		if !reflect.DeepEqual(e, expected.err) {
			t.Fatal(e, "not equal to", expected.err)
		} else if !reflect.DeepEqual(d, expected.deps) {
			t.Fatal(d, "not equal to", expected.deps)
		} else if !reflect.DeepEqual(i, expected.issues) {
			t.Fatal(i, "not equal to", expected.issues)
		}
	}
}

func setupRequirementsTxt1() {
	requirementsTxtTestData["1"] = `
click==6.6
numpy==1.14.0
pytides
`
	requirementsTxtTestResults["1"] = ResolveResult{
		deps:   d.Dependencies{d.NewDependency("click", "6.6", l.Python), d.NewDependency("numpy", "1.14.0", l.Python), d.NewDependency("pytides", "", l.Python)},
		issues: i.Issues{i.NewWeakVersion("pytides", "", "")},
		err:    nil,
	}
}

func setupRequirementsTxt2() {
	requirementsTxtTestData["2"] = `
click==6.6
git+https://github.com/happy/place.git@v0.1.8#egg=some-thing
git+https://github.com/mozilla/elasticutils.git#egg=elasticutils
pytides
`
	requirementsTxtTestResults["2"] = ResolveResult{
		deps:   d.Dependencies{d.NewDependency("click", "6.6", l.Python), d.NewDependency("place", "v0.1.8", l.Python), d.NewDependency("elasticutils", "", l.Python), d.NewDependency("pytides", "", l.Python)},
		issues: i.Issues{i.NewWeakVersion("pytides", "", "")},
		err:    nil,
	}
}
