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

	"github.com/venicegeo/vzutil-versioning/single/project/util"
)

func runAgainst(name string, gitUrl string, t *testing.T) {
	var ret util.CmdRet
	if ret = util.RunCommand("git", "clone", gitUrl); ret.Err != nil {
		t.Error(ret.Err)
	}
	defer func() {
		if ret = util.RunCommand("rm", "-rf", name); ret.Err != nil {
			t.Error(ret.Err)
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
	runAgainst("beachfront-py", "https://github.com/venicegeo/beachfront-py", t)
	runAgainst("bfalg-ndwi", "https://github.com/venicegeo/bfalg-ndwi", t)
}

func TestGo(t *testing.T) {
	runAgainst("pz-logger", "https://github.com/venicegeo/pz-logger", t)
	runAgainst("pz-workflow", "https://github.com/venicegeo/pz-workflow", t)
}

func TestJava(t *testing.T) {
	runAgainst("pz-gateway", "https://github.com/venicegeo/pz-gateway", t)
	runAgainst("bf-api", "https://github.com/venicegeo/bf-api", t)
}

func TestJavaScript(t *testing.T) {
	runAgainst("pz-sak", "https://github.com/venicegeo/pz-sak", t)
}
