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
	"testing"

	d "github.com/venicegeo/vzutil-versioning/common/dependency"
	i "github.com/venicegeo/vzutil-versioning/common/issue"
	l "github.com/venicegeo/vzutil-versioning/common/language"
)

func TestRequirementsTxt(t *testing.T) {
	addTest("requirements_txt", `
click==6.6
git+https://github.com/happy/place.git@v0.1.8#egg=some-thing
git+https://github.com/mozilla/elasticutils.git#egg=elasticutils
pytides
`, ResolveResult{
		deps:   d.Dependencies{d.NewDependency("click", "6.6", l.Python), d.NewDependency("elasticutils", "", l.Python), d.NewDependency("place", "v0.1.8", l.Python), d.NewDependency("pytides", "", l.Python)},
		issues: i.Issues{i.NewWeakVersion("pytides", "", "")},
		err:    nil,
	}, resolver.ResolveRequirementsTxt)

	addTest("requirements_txt", `
click==6.6
#comment
kcilc>=0.6
`, ResolveResult{
		deps:   d.Dependencies{d.NewDependency("click", "6.6", l.Python), d.NewDependency("kcilc", "0.6", l.Python)},
		issues: i.Issues{i.NewWeakVersion("kcilc", "0.6", ">=")},
		err:    nil,
	}, resolver.ResolveRequirementsTxt)

	run("requirements_txt", t)

}
