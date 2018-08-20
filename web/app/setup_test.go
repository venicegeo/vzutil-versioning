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

package app

import (
	"testing"

	"github.com/venicegeo/vzutil-versioning/web/es"
)

func TestProjects(t *testing.T) {
	if resp, err := testApp.index.PostData(ProjectType, "project1", es.Project{"project1", "Project One"}); err != nil {
		t.Error("Could not create project:", err.Error())
	} else if !resp.Created {
		t.Error("Project could not be created for unknown reason")
	}

	if resp, err := testApp.index.PostData(ProjectType, "project2", es.Project{"project2", "Project Two"}); err != nil {
		t.Error("Could not create project:", err.Error())
	} else if !resp.Created {
		t.Error("Project could not be created for unknown reason")
	}

	if proj, err := testApp.rtrvr.GetProject("project1"); err != nil {
		t.Error("Could not get project:", err.Error())
	} else {
		if proj.Name != "project1" {
			t.Error("Name wrong:", proj.Name)
		} else if proj.DisplayName != "Project One" {
			t.Error("Display name wrong:", proj.DisplayName)
		}
	}

	if projects, err := testApp.rtrvr.GetAllProjects(); err != nil {
		t.Error("Could not get all projects:", err.Error())
	} else {
		if len(projects) != 2 {
			t.Error("Length of projects", len(projects))
		}
	}
}
