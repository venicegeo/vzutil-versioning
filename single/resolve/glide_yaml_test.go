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

func TestGlideYaml(t *testing.T) {
	addTest("glide_yaml", `
package: some/cool/place
import:
  - package: dep_one
    version: abc
  - package: dep_two
    version: 1.3
    subpackages:
    - dont_include
testImport:
  - package: dep_three
`, ResolveResult{
		deps:   d.Dependencies{d.NewDependency("dep_one", "abc", l.Go), d.NewDependency("dep_three", "", l.Go), d.NewDependency("dep_two", "1.3", l.Go)},
		issues: i.Issues{i.NewMissingVersion("dep_three")},
		err:    nil,
	}, resolver.ResolveGlideYaml)

	run("glide_yaml", t)

}
