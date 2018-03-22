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
	"fmt"
	"reflect"
	"strings"

	lan "github.com/venicegeo/vzutil-versioning/common/language"
)

type GenericDependency struct {
	name     string
	version  string
	language lan.Language
}

func NewGenericDependency(name, version string, language lan.Language) *GenericDependency {
	return &GenericDependency{name, version, language}
}
func NewGenericDependencyStr(dep string) *GenericDependency {
	parts := strings.Split(dep, ":")
	for i, p := range parts {
		parts[i] = strings.ToLower(p)
	}
	switch len(parts) {
	case 1:
		return &GenericDependency{parts[0], "Unknown", lan.Unknown}
	case 2:
		return &GenericDependency{parts[0], parts[1], lan.Unknown}
	case 3:
		return &GenericDependency{parts[0], parts[1], lan.GetLanguage(parts[2])}
	default:
		panic(fmt.Sprintf("Bad dep split. Line %s was split into %#v", dep, parts))
	}
	if len(parts) == 1 {
		parts = append(parts, "Unknown")
	}
	return &GenericDependency{parts[0], parts[1], lan.Unknown}
}

func (d *GenericDependency) GetName() string {
	return d.name
}
func (d *GenericDependency) SetName(name string) {
	d.name = name
}
func (d *GenericDependency) GetVersion() string {
	return d.version
}
func (d *GenericDependency) SetVersion(version string) {
	d.version = version
}
func (d *GenericDependency) GetLanguage() lan.Language {
	return d.language
}
func (d *GenericDependency) SimpleEquals(dep *GenericDependency) bool {
	return strings.EqualFold(d.name, dep.name) && strings.EqualFold(d.version, dep.version)
}
func (d *GenericDependency) DeepEquals(dep *GenericDependency) bool {
	return strings.EqualFold(d.name, dep.name) && strings.EqualFold(d.version, dep.version) && strings.EqualFold(d.language.String(), dep.language.String())
}
func (dep *GenericDependency) String() string {
	return dep.name + ":" + dep.version
}
func (dep *GenericDependency) Clone() *GenericDependency {
	res := &GenericDependency{}
	reflect.ValueOf(res).Elem().Set(reflect.ValueOf(dep).Elem())
	return res
}
func (dep *GenericDependency) FullString() string {
	return dep.name + ":" + dep.version + ":" + dep.language.String()
}

func RemoveExactDuplicates(deps *[]*GenericDependency) (dups []*GenericDependency) {
	found := map[string]bool{}
	j := 0
	for i, x := range *deps {
		if !found[x.FullString()] {
			found[x.FullString()] = true
			(*deps)[j] = (*deps)[i]
			j++
		} else {
			dups = append(dups, x)
		}
	}
	filtered := make([]*GenericDependency, len(found), len(found))
	i := 0
	for k, _ := range found {
		filtered[i] = NewGenericDependencyStr(k)
		i++
	}
	*deps = filtered
	return dups
}
