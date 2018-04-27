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

import "html/template"

type HtmlCollection struct {
	html []HtmlInter
}

func NewHtmlCollection(items ...HtmlInter) *HtmlCollection {
	return &HtmlCollection{items}
}

func (h *HtmlCollection) Add(item HtmlInter) {
	h.html = append(h.html, item)
}
func (h *HtmlCollection) Template() template.HTML {
	return template.HTML(h.String())
}
func (h *HtmlCollection) String() string {
	res := ""
	for _, ht := range h.html {
		res += ht.String()
	}
	return res
}
