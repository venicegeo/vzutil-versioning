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
package dependency

import (
	"regexp"
	"strings"

	lan "github.com/venicegeo/vzutil-versioning/common/language"
)

type GenericDependencyMap map[string]GenericDependencies
type GenericDependencies []*GenericDependency

//func SwapShaVersions(shaStore sha.ShaStore, depss ...*GenericDependencies) {
//	for _, deps := range depss {
//		deps.SwapShaVersions(shaStore)
//	}
//}
func CondenseBundles(bundles map[string][]string, depss ...*GenericDependencies) {
	for _, deps := range depss {
		deps.CondenseBundles(bundles)
	}
}
func RemoveExceptions(exceptions []string, depss ...*GenericDependencies) {
	for _, deps := range depss {
		deps.RemoveExceptions(exceptions)
	}
}
func RemoveExactDuplicates(depss ...*GenericDependencies) {
	for _, deps := range depss {
		deps.RemoveExactDuplicates()
	}
}

func (d *GenericDependencies) Add(deps ...*GenericDependency) {
	*d = append(*d, deps...)

}
func (d *GenericDependencies) GetPrintString() (str string) {
	for _, dep := range *d {
		str += dep.String() + "\n"
	}
	str = strings.TrimSpace(str)
	return str
}
func (deps *GenericDependencies) SString() []string {
	res := []string{}
	for _, dep := range *deps {
		res = append(res, dep.String())
	}
	return res
}
func (deps *GenericDependencies) Clone() GenericDependencies {
	res := make([]*GenericDependency, len(*deps))
	for i, dep := range *deps {
		res[i] = dep.Clone()
	}
	return GenericDependencies(res)
}
func (deps *GenericDependencies) RemoveExactDuplicates() (dups GenericDependencies) {
	found := map[string]bool{}
	j := 0
	for i, x := range *deps {
		if !found[x.FullString()] {
			found[x.FullString()] = true
			(*deps)[j] = (*deps)[i]
			j++
		} else {
			dups.Add(x)
		}
	}
	filtered := GenericDependencies{}
	for k, _ := range found {
		filtered.Add(NewGenericDependencyStr(k))
	}
	*deps = filtered
	return dups
}
func (deps *GenericDependencies) RemoveDuplicatesByProject() {
	res := GenericDependencies{}
	found := false
	for _, dep := range *deps {
		found = false
		for _, resDep := range res {
			if resDep.LanguageEquals(dep) {
				found = true
				break
			}
		}
		if !found {
			res.Add(dep)
		}
	}
	*deps = res
}

//func (deps *GenericDependencies) SwapShaVersions(shaStore sha.ShaStore) {
//	for _, dep := range *deps {
//		if dep.language != lan.Go {
//			continue
//		}
//		for _, store := range shaStore {
//			if store.Sha == dep.version {
//				dep.version = store.Version
//				break
//			}
//		}
//	}
//}
func (deps *GenericDependencies) CondenseBundles(bundles map[string][]string) {
	sortedByProject := map[string]GenericDependencies{}
	langs := map[string]lan.Language{}
	for _, dep := range *deps {
		if _, ok := sortedByProject[dep.project]; !ok {
			sortedByProject[dep.project] = GenericDependencies{}
		}
		sortedByProject[dep.project] = append(sortedByProject[dep.project], dep)
		if _, ok := langs[dep.project]; !ok {
			langs[dep.project] = dep.language
		}
	}
	res := GenericDependencies{}
	for projectName, depends := range sortedByProject {
		bundleCount := map[string]map[string]int{}
		refined := GenericDependencies{}
		for bundleName, _ := range bundles {
			bundleCount[bundleName] = map[string]int{}
		}
		for _, dep := range depends {
			found := false
			for bundleName, depNames := range bundles {
				for _, depName := range depNames {
					if dep.name == depName {
						bundleCount[bundleName][dep.version]++
						found = true
					}
				}
			}
			if !found {
				refined.Add(dep)
			}
		}
		for bundleName, v := range bundleCount {
			for version, count := range v {
				if count > 0 {
					refined.Add(NewGenericDependency(bundleName, version, projectName, langs[projectName]))
				}
			}
		}
		res = append(res, refined...)
		//		sortedByProject[projectName] = refined
	}
	*deps = res
}
func (deps *GenericDependencies) RemoveExceptions(exceptions []string) {
	re := []*regexp.Regexp{}
	for _, str := range exceptions {
		re = append(re, regexp.MustCompile(str))
	}
	res := GenericDependencies{}
	add := true
	for _, dep := range *deps {
		add = true
		for _, r := range re {
			if r.MatchString(dep.name) {
				add = false
				break
			}
		}
		if add {
			res.Add(dep)
		}
	}
	*deps = res
}
