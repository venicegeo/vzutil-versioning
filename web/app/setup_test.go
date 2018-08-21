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

	"github.com/stretchr/testify/assert"
	"github.com/venicegeo/pz-gocommon/elasticsearch/elastic-5-api"
	"github.com/venicegeo/vzutil-versioning/web/es"
)

func TestAddProjects(t *testing.T) {
	assert := assert.New(t)

	resp, err := testApp.index.PostData(ProjectType, "project1", es.Project{"project1", "Project One"})
	assert.Nil(err)
	assert.True(resp.Created)

	resp, err = testApp.index.PostData(ProjectType, "project2", es.Project{"project2", "Project Two"})
	assert.Nil(err)
	assert.True(resp.Created)

	proj, err := testApp.rtrvr.GetProject("project1")
	assert.Nil(err)
	assert.Equal("project1", proj.Name)
	assert.Equal("Project One", proj.DisplayName)

	projects, err := testApp.rtrvr.GetAllProjects()
	assert.Nil(err)
	assert.Len(projects, 2)
}

func TestAddRepositories(t *testing.T) {
	assert := assert.New(t)

	test := func(resp *elastic.IndexResponse, err error) {
		assert.Nil(err)
		assert.True(resp.Created)
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
	assert.Nil(err)
	assert.Len(proj1repos, 2)

	proj2repo, project2, err := testApp.rtrvr.GetRepository("venicegeo/pz-gateway", "project2")
	assert.Nil(err)
	assert.NotNil(project2)
	assert.Equal("venicegeo/pz-gateway", proj2repo.RepoFullname)
	assert.Equal("project2", proj2repo.ProjectName)

	projects, err := testApp.rtrvr.GetAllProjectNamesUsingRepository("venicegeo/pz-gateway")
	assert.Nil(err)
	assert.Len(projects, 2)

	projects, err = testApp.rtrvr.GetAllProjectNamesUsingRepository("venicegeo/bfalg-ndwi")
	assert.Nil(err)
	assert.Len(projects, 1)
}

func TestFire(t *testing.T) {
	proj1gateway, _, _ := testApp.rtrvr.GetRepository("venicegeo/pz-gateway", "project1")
	proj2gateway, _, _ := testApp.rtrvr.GetRepository("venicegeo/pz-gateway", "project2")
	ndwi, _, _ := testApp.rtrvr.GetRepository("venicegeo/bfalg-ndwi", "project1")
	testApp.ff.FireRequest(&SingleRunnerRequest{proj1gateway, "47bd5b191a28b637e44170cf93a50b0a2b4075f7", "refs/heads/master"})
	testApp.ff.FireRequest(&SingleRunnerRequest{proj2gateway, "14d63d433469009a1acfecaf32d37087e16e528b", "refs/heads/master"})
	testApp.ff.FireRequest(&SingleRunnerRequest{ndwi, "2997b89620dc34547c3c9c2ba755a9e769814d5e", "refs/heads/Production"})
	start := time.Now()
	time.Sleep(time.Second * 2)
	for testApp.wrkr.JobsInSystem() != 0 {
		time.Sleep(time.Second * 2)
	}
	if (time.Now().Unix() - start.Unix()) > int64(time.Second*5) {
		t.Error("Something ran too fast")
	}
}

func TestGetRepositories(t *testing.T) {
	assert := assert.New(t)

	repos, err := testApp.rtrvr.ListRepositories()
	assert.Nil(err)
	assert.Len(repos, 2)

	project, _ := testApp.rtrvr.GetProject("project1")
	refs, err := project.GetAllRefs()
	assert.Nil(err)
	assert.Len(refs, 2)

	repo, err := project.GetRepository("venicegeo/pz-gateway")
	assert.Nil(err)
	refs, err = repo.GetAllRefs()
	assert.Nil(err)
	assert.Len(refs, 1)

	project, _ = testApp.rtrvr.GetProject("project2")
	refs, err = project.GetAllRefs()
	assert.Nil(err)
	assert.Len(refs, 1)
}

func TestGetScans(t *testing.T) {
	assert := assert.New(t)

	proj1, _ := testApp.rtrvr.GetProject("project1")
	proj2, _ := testApp.rtrvr.GetProject("project2")

	scan, found, err := proj1.ScanBySha("47bd5b191a28b637e44170cf93a50b0a2b4075f7")
	assert.Nil(err)
	assert.True(found)
	assert.Equal("project1", scan.Project)
	assert.Equal("venicegeo/pz-gateway", scan.RepoFullname)

	scan, found, err = proj2.ScanBySha("47bd5b191a28b637e44170cf93a50b0a2b4075f7")
	assert.Nil(err)
	assert.False(found)
	assert.Nil(scan)

}
