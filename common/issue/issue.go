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
package issue

import (
	"fmt"
)

type Issues []issue
type issue string

func (is Issues) Len() int      { return len(is) }
func (is Issues) Swap(i, j int) { is[i], is[j] = is[j], is[i] }
func (is Issues) Less(i, j int) bool {
	return is[i] < is[j]
}
func (i *Issues) SSlice() []string {
	res := make([]string, len(*i), len(*i))
	for i, v := range *i {
		res[i] = v.String()
	}
	return res
}

func (i *issue) String() string {
	return string(*i)
}

func NewIssue(format string, a ...interface{}) issue {
	return issue(fmt.Sprintf(format, a...))
}

func NewUnusedVariable(varName, value string) issue {
	return issue(fmt.Sprintf("Unused variable [${%s}] with value [%s]", varName, value))
}

func NewVersionMismatch(packag, verA, verB string) issue {
	if verA == "" {
		verA = "NONE"
	}
	if verB == "" {
		verB = "NONE"
	}
	return issue(fmt.Sprintf("Version mismatch on package [%s]: [%s] [%s]", packag, verA, verB))
}

func NewUnknownSha(name, sha string) issue {
	return issue(fmt.Sprintf("Unknown sha [%s] for package [%s]", sha, name))
}

func NewWeakVersion(name, version, tag string) issue {
	return issue(fmt.Sprintf("Version [%s] on package [%s] is not definite. Tag: [%s]", version, name, tag))
}

func NewMissingVersion(name string) issue {
	return issue(fmt.Sprintf("Package [%s] is missing a version", name))
}
