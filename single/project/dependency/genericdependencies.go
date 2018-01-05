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
	"strings"
)

type GenericDependencies []*GenericDependency

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
