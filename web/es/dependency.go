// Copyright 2018, RadiantBlue Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package es

import (
	u "github.com/venicegeo/vzutil-versioning/web/util"
)

type DependencySort []Dependency

func (d DependencySort) Len() int      { return len(d) }
func (d DependencySort) Swap(i, j int) { d[i], d[j] = d[j], d[i] }
func (d DependencySort) Less(i, j int) bool {
	if d[i].Language != d[j].Language {
		return d[i].Language < d[j].Language
	} else {
		return d[i].Name < d[j].Name
	}
}

type Dependency struct {
	Name     string `json:"name"`
	Version  string `json:"version"`
	Language string `json:"language"`
}

func (d *Dependency) GetHashSum() string {
	return u.Hash(u.Format("%s:%s:%s", d.Name, d.Version, d.Language))
}

func (d *Dependency) String() string {
	return u.Format("%s:%s:%s", d.Name, d.Version, d.Language)
}
