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

package structs

type GitWebhook struct {
	Zen        string        `json:"zen"`
	Ref        string        `json:"ref"`
	BeforeSha  string        `json:"before"`
	AfterSha   string        `json:"after"`
	Repository GitRepository `json:"repository"`
	Timestamp  int64
}
type GitRepository struct {
	Id       int64  `json:"id"`
	Name     string `json:"name"`
	FullName string `json:"full_name"`
	HtmlUrl  string `json:"html_url"`
	Url      string `json:"url"`
}
