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
package project

import (
	"fmt"
	"testing"

	"github.com/venicegeo/vzutil-versioning/common/dependency"
	"github.com/venicegeo/vzutil-versioning/single/project/issue"
)

func TestIngestPython1(t *testing.T) {
	p := &Project{
		FolderName:     "python-test1-name",
		FolderLocation: "",
		Sha:            "python-test1-sha",
		Dependencies:   []*dependency.GenericDependency{},
		DepLocations:   []string{"environment.yml"},
		Issues:         []*issue.Issue{},
	}
	i := &Ingester{&MockFileReader{}}
	errs := i.IngestProject(p)
	if len(errs) != 0 {
		t.Error(errs[0])
	}
	for _, d := range p.Dependencies {
		fmt.Println(d.String())
	}
}
