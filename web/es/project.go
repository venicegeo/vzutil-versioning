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

type Project struct {
	FullName string   `json:"full_name"`
	Name     string   `json:"name"`
	TagShas  []TagSha `json:"tag_shas"`
	Refs     []Ref    `json:"refs"`
}

type Ref struct {
	Name         string         `json:"name"`
	WebhookOrder []string       `json:"webhook_order"`
	Entries      []ProjectEntry `json:"entries"`
}

type ProjectEntry struct {
	Sha            string   `json:"sha"`
	EntryReference string   `json:"entry_reference"`
	Dependencies   []string `json:"dependencies"`
}

type TagSha struct {
	Tag string `json:"tag"`
	Sha string `json:"sha"`
}

func NewProject(fullName, name string) *Project {
	return &Project{
		FullName: fullName,
		Name:     name,
		TagShas:  []TagSha{},
		Refs:     []Ref{}}
}

func NewRef(refName string) *Ref {
	return &Ref{
		Name:         refName,
		WebhookOrder: []string{},
		Entries:      []ProjectEntry{},
	}
}

func (p *Project) GetShaFromTag(tag string) (string, bool) {
	for _, ts := range p.TagShas {
		if ts.Tag == tag {
			return ts.Sha, true
		}
	}
	return "", false
}
func (p *Project) GetTagFromSha(sha string) (string, bool) {
	for _, ts := range p.TagShas {
		if ts.Sha == sha {
			return ts.Tag, true
		}
	}
	return "", false
}

func (p *Project) GetEntry(sha string) (ref Ref, entry ProjectEntry, found bool) {
	for _, ref := range p.Refs {
		if entry, found = ref.GetEntry(sha); found {
			return ref, entry, found
		}
	}
	return ref, entry, false
}

func (r *Ref) MustGetEntry(sha string) (entry ProjectEntry) {
	for _, e := range r.Entries {
		if e.Sha == sha {
			entry = e
			break
		}
	}
	return entry
}

func (r *Ref) GetEntry(sha string) (entry ProjectEntry, found bool) {
	for _, e := range r.Entries {
		if e.Sha == sha {
			entry = e
			found = true
			break
		}
	}
	return entry, found
}
