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
	"time"

	"github.com/venicegeo/pz-gocommon/elasticsearch/elastic-5-api"
	"github.com/venicegeo/vzutil-versioning/web/es"
)

func TestAddProjects(t *testing.T) {
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

func TestAddRepositories(t *testing.T) {
	test := func(resp *elastic.IndexResponse, err error) {
		if err != nil {
			t.Error("Could not create repository:", err.Error())
		} else if !resp.Created {
			t.Error("Repository could not be created")
		}
	}
	test(testApp.index.PostData(ProjectEntryType, "", es.ProjectEntry{
		ProjectName:    "project1",
		RepoFullname:   "venicegeo/pz-gateway",
		DependencyInfo: es.ProjectEntryDependencyInfo{"venicegeo/pz-gateway", es.IncomingSha, "", []string{"/pom.xml"}},
	}))

	test(testApp.index.PostData(ProjectEntryType, "", es.ProjectEntry{
		ProjectName:    "project1",
		RepoFullname:   "venicegeo/bfalg-ndwi",
		DependencyInfo: es.ProjectEntryDependencyInfo{"venicegeo/venicegeo-conda-recipes", es.ExactSha, "e82b5ca6388263324301e90ead3fbcf4cd5d360a", []string{"/recipes/bfalg-ndwi/meta.yaml"}},
	}))

	test(testApp.index.PostData(ProjectEntryType, "", es.ProjectEntry{
		ProjectName:    "project2",
		RepoFullname:   "venicegeo/pz-gateway",
		DependencyInfo: es.ProjectEntryDependencyInfo{"venicegeo/pz-gateway", es.IncomingSha, "", []string{"/pom.xml"}},
	}))

	proj1, _ := testApp.rtrvr.GetProject("project1")
	proj1repos, err := proj1.GetAllRepositories()
	if err != nil {
		t.Error("Unable to get project1 repos:", err.Error())
	} else if len(proj1repos) != 2 {
		t.Error("Proj1 didnt return 2 repos:", len(proj1repos))
	}
	proj2repo, project2, err := testApp.rtrvr.GetRepository("venicegeo/pz-gateway", "project2")
	if err != nil {
		t.Error("Unable to get project2 repo:", err.Error())
	} else if project2 == nil {
		t.Error("Unable to get the project for project2 repo")
	} else if proj2repo.RepoFullname != "venicegeo/pz-gateway" {
		t.Error("Repo name is wrong:", proj2repo.RepoFullname)
	} else if proj2repo.ProjectName != "project2" {
		t.Error("Project name is wrong:", proj2repo.ProjectName)
	}
	// Not supported
	//	if projects, err := testApp.rtrvr.GetAllProjectNamesUsingRepository("venicegeo/pz-gateway"); err != nil {
	//		t.Error("Error getting project using repository:", err.Error())
	//	} else if len(projects) != 2 {
	//		t.Error("did not return 2 projects:", len(projects))
	//	}
	//	if projects, err := testApp.rtrvr.GetAllProjectNamesUsingRepository("venicegeo/bfalg-ndwi"); err != nil {
	//		t.Error("Error get project using repository", err.Error())
	//	} else if len(projects) != 1 {
	//		t.Error("did not return 1 project", len(projects))
	//	}
}

func TestFire(t *testing.T) {
	proj1gateway, _, _ := testApp.rtrvr.GetRepository("venicegeo/pz-gateway", "project1")
	proj2gateway, _, _ := testApp.rtrvr.GetRepository("venicegeo/pz-gateway", "project2")
	ndwi, _, _ := testApp.rtrvr.GetRepository("venicegeo/bfalg-ndwi", "project1")
	testApp.ff.FireRequest(&SingleRunnerRequest{proj1gateway, "47bd5b191a28b637e44170cf93a50b0a2b4075f7", "refs/heads/master"})
	testApp.ff.FireRequest(&SingleRunnerRequest{proj2gateway, "14d63d433469009a1acfecaf32d37087e16e528b", "refs/heads/master"})
	testApp.ff.FireRequest(&SingleRunnerRequest{ndwi, "2997b89620dc34547c3c9c2ba755a9e769814d5e", "refs/heads/Production"})
	time.Sleep(time.Minute)
}
