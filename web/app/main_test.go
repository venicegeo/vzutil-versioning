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
	"os"
	"testing"

	"github.com/venicegeo/pz-gocommon/elasticsearch"
	j "github.com/venicegeo/pz-gocommon/gocommon"
	"github.com/venicegeo/vzutil-versioning/web/es"
)

var testApp *Application

func TestMain(m *testing.M) {
	index := elasticsearch.NewMockIndex("versioning_tool")
	index.Create("")
	index.SetMapping(RepositoryEntry_QType, j.JsonString(RepositoryDependencyScanMapping))
	index.SetMapping(Difference_QType, j.JsonString(DifferenceMapping))
	index.SetMapping(ProjectEntryType, j.JsonString(es.ProjectEntryMapping))
	index.SetMapping(Project_QType, j.JsonString(es.ProjectMapping))

	testApp = NewApplication(index, "../single", "../compare", "../templates/", false)
	testApp.StartInternals()

	os.Exit(m.Run())
}
