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

func CompareSimple(source, expected *GenericDependencies) (*GenericDependencies, *GenericDependencies, *GenericDependencies) {
	equals := func(a *GenericDependency, b *GenericDependency) bool {
		return a.SimpleEquals(b)
	}
	return compare(source, expected, equals)
}
func CompareByLanguage(source, expected *GenericDependencies) (*GenericDependencies, *GenericDependencies, *GenericDependencies) {
	equals := func(a *GenericDependency, b *GenericDependency) bool {
		return a.LanguageEquals(b)
	}
	return compare(source, expected, equals)
}
func CompareByProject(source, expected *GenericDependencies) (*GenericDependencies, *GenericDependencies, *GenericDependencies) {
	equals := func(a *GenericDependency, b *GenericDependency) bool {
		return a.ProjectEquals(b)
	}
	return compare(source, expected, equals)
}
func compare(source, expected *GenericDependencies, equals func(*GenericDependency, *GenericDependency) bool) (*GenericDependencies, *GenericDependencies, *GenericDependencies) {
	sourceMissing := &GenericDependencies{}
	sourceExtra := &GenericDependencies{}
	sourceSame := &GenericDependencies{}

	scrubChan := make(chan bool, 3)

	scrub := func(source, test, dest *GenericDependencies) {
		found := false
		for _, sourceDep := range *source {
			found = false
			for _, testDep := range *test {
				if equals(sourceDep, testDep) {
					found = true
					sourceSame.Add(sourceDep.Clone())
					break
				}
			}
			if !found {
				dest.Add(sourceDep.Clone())
			}
		}
		scrubChan <- true
	}

	//TODO routines

	scrub(source, expected, sourceExtra)
	scrub(expected, source, sourceMissing)
	//sourceSame.RemoveDuplicatesByProject()

	for i := 0; i < len(scrubChan); i++ {
		<-scrubChan
	}

	return sourceMissing, sourceExtra, sourceSame
}
