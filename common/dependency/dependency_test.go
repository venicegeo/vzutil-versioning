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
package dependency

import (
	"fmt"
	"testing"

	"github.com/venicegeo/vzutil-versioning/project/language"
)

var testName, testVersion, testProject, testUnknown = "foo", "1.0.0", "bar", "Unknown"
var testLanguage = language.Go

func TestConstructors(t *testing.T) {
	failIf := func(b bool) {
		if b {
			t.FailNow()
		}
	}
	dep := NewGenericDependency(testName, testVersion, testProject, testLanguage)
	failIf(dep.name != testName || dep.version != testVersion || dep.project != testProject || dep.language != testLanguage)

	dep = NewGenericDependencyStr(fmt.Sprintf("%s", testName))
	failIf(dep.name != testName || dep.version != testUnknown || dep.project != testUnknown || dep.language != language.Unknown)

	dep = NewGenericDependencyStr(fmt.Sprintf("%s:%s", testName, testVersion))
	failIf(dep.name != testName || dep.version != testVersion || dep.project != testUnknown || dep.language != language.Unknown)

	dep = NewGenericDependencyStr(fmt.Sprintf("%s:%s:%s", testName, testVersion, testProject))
	failIf(dep.name != testName || dep.version != testVersion || dep.project != testProject || dep.language != language.Unknown)

	dep = NewGenericDependencyStr(fmt.Sprintf("%s:%s:%s:%s", testName, testVersion, testProject, "gostack"))
	failIf(dep.name != testName || dep.version != testVersion || dep.project != testProject || dep.language != testLanguage)

	dep = NewGenericDependencyStr(fmt.Sprintf("%s:%s:%s:%s", testName, testVersion, testProject, "go"))
	failIf(dep.name != testName || dep.version != testVersion || dep.project != testProject || dep.language != testLanguage)

	failIf(!NewGenericDependency(testName, testVersion, "a", testLanguage).SimpleEquals(NewGenericDependency(testName, testVersion, "b", language.Unknown)))
	failIf(NewGenericDependency(testName, testVersion, "a", testLanguage).SimpleEquals(NewGenericDependency(testName, "2.0.1", "b", language.Unknown)))
}
