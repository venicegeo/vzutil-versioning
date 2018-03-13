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

	"github.com/venicegeo/vzutil-versioning/project/util"
)

func runAgainst(name string, gitUrl string, t *testing.T) {
	var err error
	if _, err = util.RunCommand("git", "clone", gitUrl); err != nil {
		t.Error(err)
	}
	defer func() {
		if _, err = util.RunCommand("rm", "-rf", name); err != nil {
			t.Error(err)
		}
	}()
	project, err := NewProject(name)
	if err != nil {
		t.Error(err)
	}
	if err = Ingest(project, true); err != nil {
		t.Error(err)
	}

	fmt.Printf("### Direct dependencies found for %s version %s\n", project.FolderName, project.Sha)
	for _, s := range project.GetDependencies() {
		fmt.Printf("###   %s\n", s)
	}

}

func TestPython(t *testing.T) {
	runAgainst("bf-api", "https://github.com/venicegeo/bf-api", t)
	runAgainst("bfalg-ndwi", "https://github.com/venicegeo/bfalg-ndwi", t)
}

func TestGo(t *testing.T) {
	runAgainst("pz-logger", "https://github.com/venicegeo/pz-logger", t)
}

func TestJava(t *testing.T) {
	runAgainst("pz-gateway", "https://github.com/venicegeo/pz-gateway", t)
}

func TestJavaScript(t *testing.T) {
	runAgainst("pz-sak", "https://github.com/venicegeo/pz-sak", t)
}
