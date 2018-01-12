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

package es

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/venicegeo/pz-gocommon/elasticsearch"
)

func GetProjectById(index *elasticsearch.Index, fullName string) (*Project, error) {
	docName := strings.Replace(fullName, "/", "_", -1)
	resp, err := index.GetByID("project", docName)
	if err != nil {
		return nil, err
	}
	if !resp.Found {
		return nil, fmt.Errorf("Could not find this document: [%s]", docName)
	}
	project := &Project{}
	if err = json.Unmarshal([]byte(*resp.Source), project); err != nil {
		return nil, err
	}
	return project, nil
}
