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

import (
	"bytes"
	"fmt"
	"html/template"
	"sort"

	u "github.com/venicegeo/vzutil-versioning/web/util"
)

type HtmlAccordion struct {
	acc  map[string]string
	keys []string
}

func NewHtmlAccordion() *HtmlAccordion {
	return &HtmlAccordion{map[string]string{}, []string{}}
}

func (h *HtmlAccordion) AddItem(name string, elem fmt.Stringer) *HtmlAccordion {
	h.keys = append(h.keys, name)
	h.acc[name] = elem.String()
	return h
}
func (h *HtmlAccordion) UpdateItem(name string, elem fmt.Stringer) *HtmlAccordion {
	h.acc[name] = elem.String()
	return h
}
func (h *HtmlAccordion) AddKey(name string) *HtmlAccordion {
	h.keys = append(h.keys, name)
	return h
}

func (h *HtmlAccordion) Sort() *HtmlAccordion {
	sort.Strings(h.keys)
	return h
}

func (h *HtmlAccordion) Template() template.HTML {
	return template.HTML(h.String())
}

func (h *HtmlAccordion) String() string {
	buf := bytes.NewBufferString("")
	for i, name := range h.keys {
		buf.WriteString(u.Format("<button class=\"accordion\">%s</button>\n", name))
		buf.WriteString("\t<div class=\"panel\">\n\t\t")
		buf.WriteString(h.acc[name])
		buf.WriteString("\n\t</div>")
		if i < len(h.keys)-1 {
			buf.WriteString("\n")
		}
	}
	return buf.String()
}
