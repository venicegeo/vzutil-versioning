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

type IssuesMap map[string]*Issues
type Issues []string
type Issue string

func (i *Issue) String() string {
	return string(*i)
}

func NewIssue(format string, a ...interface{}) *Issue {
	temp := Issue(fmt.Sprintf(format, a...))
	return &temp
}

func NewUnusedVariable(varName, value string) *Issue {
	temp := Issue(fmt.Sprintf("Unused variable [${%s}] with value [%s]", varName, value))
	return &temp
}

func NewVersionMismatch(packag, verA, verB string) *Issue {
	if verA == "" {
		verA = "NONE"
	}
	if verB == "" {
		verB = "NONE"
	}
	temp := Issue(fmt.Sprintf("Version mismatch on package [%s]: [%s] [%s]", packag, verA, verB))
	return &temp
}

func NewUnknownSha(name, sha string) *Issue {
	temp := Issue(fmt.Sprintf("Unknown sha [%s] for package [%s]", sha, name))
	return &temp
}

func NewWeakVersion(name, version, tag string) *Issue {
	temp := Issue(fmt.Sprintf("Version [%s] on package [%s] is not definite. Tag: [%s]", version, name, tag))
	return &temp
}

func NewMissingVersion(name string) *Issue {
	temp := Issue(fmt.Sprintf("Package [%s] is missing a version", name))
	return &temp
}
