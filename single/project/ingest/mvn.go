/*
Copyright 2017, RadiantBlue Technologies, Inc.

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
package ingest

import (
	"sync"

	"github.com/venicegeo/vzutil-versioning/single/project/util"
)

var mvnMux = sync.Mutex{}

type MvnDependency struct {
	GroupId    string `json:"groupId"`
	ArtifactId string `json:"artifactId"`
	Packaging  string `json:"packaging"`
	Version    string `json:"version,omitempty"`
}

func GenerateMvnReport(location string) util.CmdRet {
	mvnMux.Lock()
	cmd := util.RunCommand("mvn", "-f", location, "dependency:resolve")
	if cmd.IsError() {
		tmp := util.RunCommand("ls", location)
		cmd.Stdout += "\n" + tmp.String()
	}
	mvnMux.Unlock()
	return cmd
}
