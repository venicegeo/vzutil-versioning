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

func TestPackageJson(t *testing.T) {
	addTest("package_json", `
{
	"devDependencies": {
		"karma": "42",
		"mocha": "50"
	},
	"dependencies": {
		"ol": "ok",
		"ok": "ol"
	}
}
`, ResolveResult{
		deps:   d.Dependencies{d.NewDependency("ol", "ok", l.JavaScript), d.NewDependency("ok", "ol", l.JavaScript), d.NewDependency("karma", "42", l.JavaScript), d.NewDependency("mocha", "50", l.JavaScript)},
		issues: i.Issues{},
		err:    nil,
	}, resolver.ResolvePackageJson)

	addTest("package_json", `
{
	"dependencies": {
		"babel-core": "~6.26.3"
	}
}`, ResolveResult{
		deps:   d.Dependencies{d.NewDependency("babel-core", "6.26.3", l.JavaScript)},
		issues: i.Issues{i.NewWeakVersion("babel-core", "~6.26.3", "~")},
		err:    nil,
	}, resolver.ResolvePackageJson)

	run("package_json", t)

}
