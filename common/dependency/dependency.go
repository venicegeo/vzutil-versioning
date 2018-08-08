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

const DependencyMapping string = `{
	"dynamic":"strict",
	"properties":{
		"name":{"type":"keyword"},
		"version":{"type":"keyword"},
		"language":{"type":"keyword"}
	}
}`

type Dependency struct {
	Name     string       `json:"name"`
	Version  string       `json:"version"`
	Language lan.Language `json:"language"`
}

func NewDependency(name, version string, language lan.Language) Dependency {
	return Dependency{strings.ToLower(name), strings.ToLower(version), language}
}
func NewDependencyStr(dep string) Dependency {
	parts := strings.Split(dep, ":")
	for i, p := range parts {
		parts[i] = strings.ToLower(p)
	}
	switch len(parts) {
	case 1:
		return Dependency{parts[0], "unknown", lan.Unknown}
	case 2:
		return Dependency{parts[0], parts[1], lan.Unknown}
	case 3:
		return Dependency{parts[0], parts[1], lan.GetLanguage(parts[2])}
	default:
		panic(fmt.Sprintf("Bad dep split. Line %s was split into %#v", dep, parts))
	}
	if len(parts) == 1 {
		parts = append(parts, "Unknown")
	}
	return Dependency{parts[0], parts[1], lan.Unknown}
}

func (d *Dependency) SimpleEquals(dep *Dependency) bool {
	return strings.EqualFold(d.Name, dep.Name) && strings.EqualFold(d.Version, dep.Version)
}
func (d *Dependency) DeepEquals(dep *Dependency) bool {
	return strings.EqualFold(d.Name, dep.Name) && strings.EqualFold(d.Version, dep.Version) && strings.EqualFold(d.Language.String(), dep.Language.String())
}
func (dep *Dependency) String() string {
	return dep.Name + ":" + dep.Version
}
func (dep *Dependency) Clone() *Dependency {
	res := &Dependency{}
	reflect.ValueOf(res).Elem().Set(reflect.ValueOf(dep).Elem())
	return res
}
func (dep *Dependency) FullString() string {
	return dep.Name + ":" + dep.Version + ":" + dep.Language.String()
}

func RemoveExactDuplicates(deps *Dependencies) (dups Dependencies) {
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
	filtered := make(Dependencies, len(found), len(found))
	i := 0
	for k, _ := range found {
		filtered[i] = NewDependencyStr(k)
		i++
	}
	*deps = filtered
	return dups
}

type Dependencies []Dependency

func (d Dependencies) Len() int      { return len(d) }
func (d Dependencies) Swap(i, j int) { d[i], d[j] = d[j], d[i] }
func (d Dependencies) Less(i, j int) bool {
	if d[i].Language != d[j].Language {
		return d[i].Language < d[j].Language
	} else {
		return d[i].Name < d[j].Name
	}
}
