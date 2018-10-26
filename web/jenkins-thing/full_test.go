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
package j

import (
	"log"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/venicegeo/pz-gocommon/elasticsearch"
	"github.com/venicegeo/vzutil-versioning/web/jenkins-thing/types"

	n "github.com/venicegeo/vzutil-versioning/web/jenkins-thing/nt"
)

func TestOne(t *testing.T) {

	assert := assert.New(t)
	index := elasticsearch.NewMockIndex("test")
	err := index.Create(`{
	"mappings":{
		"` + PipelineEntryType + `": ` + types.PipelineEntryMapping + `,
		"` + TargetsType + `": ` + types.TargetsMapping + `
	}
}`)
	assert.Nil(err)

	pipeline := NewPipeline("test-pipeline", "testorg/test")
	pipeline.AddBuild(NewBuild(20, "1243457879461345497784546154879451613457").AddStage(NewStage("stage 1", true).AddTarget(NewTarget("myorg", "myspace", true))))

	http := pipeline.CreateHttpMap()
	manager := NewManager(index, http)
	id, err := manager.Add("a", pipeline.GetRepoUrl(), pipeline.GetUrl())
	assert.Nil(err)

	err = manager.RunScan()
	assert.Nil(err)

	lastSuccesses, err := manager.GetLastSuccesses(id)
	assert.Nil(err)
	log.Println(lastSuccesses)
	//	allSuccesses, err := manager.GetAllSuccesses(id)
	//	assert.Nil(err)
	//	log.Println(allSuccesses)
}

func TestTwo(t *testing.T) {
	index, err := elasticsearch.NewIndex2("http://localhost:9200", "", "", "test", `{
	"mappings":{
		"`+PipelineEntryType+`": `+t.PipelineEntryMapping+`,
		"`+TargetsType+`": `+t.TargetsMapping+`
	}
}`)
	if err != nil {
		log.Fatalln(err)
	}
	log.Println(index.IndexName())

	manager := NewManager(index, &n.TestHTTP_FS{"/home/ubuntu/Desktop/go/src/github.com/venicegeo/vzutil-versioning/web/jenkins-thing/http_test"}, url)
	//TODO add test here

	manager.RunScan()

	log.Println(manager.GetLastSuccesses(""))
	log.Println(manager.GetAllSuccesses(""))

	index.Delete()
}
