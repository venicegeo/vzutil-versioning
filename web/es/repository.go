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
	"strings"
)

type Repository struct {
	FullName string `json:"full_name"`
	Name     string `json:"name"`
	Refs     []*Ref `json:"refs"`
}

type Ref struct {
	Name         string            `json:"name"`
	WebhookOrder []string          `json:"webhook_order"`
	Entries      []RepositoryEntry `json:"entries"`
}

type RepositoryEntry struct {
	Sha            string   `json:"sha"`
	Timestamp      int64    `json:"timestamp"`
	EntryReference string   `json:"entry_reference"`
	Dependencies   []string `json:"dependencies"`
}

func NewRepository(fullName, name string) *Repository {
	return &Repository{
		FullName: fullName,
		Name:     name,
		Refs:     []*Ref{}}
}

func NewRef(refName string) *Ref {
	return &Ref{
		Name:         refName,
		WebhookOrder: []string{},
		Entries:      []RepositoryEntry{},
	}
}

func (p *Repository) GetShaFromTag(tag string) (string, bool) {
	ref := "refs/tags/" + tag
	for _, r := range p.Refs {
		if r.Name == ref {
			return r.Entries[0].Sha, true
		}
	}
	return "", false
}
func (p *Repository) GetTagFromSha(sha string) (string, bool) {
	for _, r := range p.Refs {
		for _, entry := range r.Entries {
			if entry.Sha == sha {
				return strings.TrimPrefix(r.Name, "/refs/tags/"), true
			}
		}
	}
	return "", false
}

func (p *Repository) GetEntry(sha string) (ref *Ref, entry RepositoryEntry, found bool) {
	for _, ref := range p.Refs {
		if entry, found = ref.GetEntry(sha); found {
			return ref, entry, found
		}
	}
	return ref, entry, false
}

func (r *Ref) MustGetEntry(sha string) (entry RepositoryEntry) {
	for _, e := range r.Entries {
		if e.Sha == sha {
			entry = e
			break
		}
	}
	return entry
}

func (r *Ref) GetEntry(sha string) (entry RepositoryEntry, found bool) {
	for _, e := range r.Entries {
		if e.Sha == sha {
			entry = e
			found = true
			break
		}
	}
	return entry, found
}
