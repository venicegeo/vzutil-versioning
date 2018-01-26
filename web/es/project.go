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
)

type Project struct {
	FullName     string   `json:"full_name"`
	Name         string   `json:"name"`
	LastSha      string   `json:"last_sha"`
	WebhookOrder []string `json:"webhook_order"`
	TagShas      string   `json:"tag_shas"`
	Entries      string   `json:"entries"`
}

type ProjectEntries map[string]ProjectEntry

type ProjectEntry struct {
	EntryReference string   `json:"entry_reference"`
	Dependencies   []string `json:"dependencies"`
}

func NewProject(fullName, name string) *Project {
	temp := ProjectEntries{}
	temp2 := map[string]string{}
	dat, _ := json.Marshal(temp)
	dat2, _ := json.Marshal(temp2)
	return &Project{fullName, name, "", []string{}, string(dat2), string(dat)}
}

func (p *Project) GetEntries() (*ProjectEntries, error) {
	var entries ProjectEntries
	return &entries, json.Unmarshal([]byte(p.Entries), &entries)
}

func (p *Project) SetEntries(entries *ProjectEntries) error {
	dat, err := json.Marshal(entries)
	if err != nil {
		return err
	}
	p.Entries = string(dat)
	return nil
}

func (p *Project) GetTagShas() (*map[string]string, error) {
	var shas map[string]string
	return &shas, json.Unmarshal([]byte(p.TagShas), &shas)
}

func (p *Project) SetTagShas(shas *map[string]string) error {
	dat, err := json.Marshal(shas)
	if err != nil {
		return err
	}
	p.TagShas = string(dat)
	return nil
}
