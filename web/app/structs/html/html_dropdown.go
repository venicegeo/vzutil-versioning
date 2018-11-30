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
	"html/template"

	u "github.com/venicegeo/vzutil-versioning/web/util"
)

type option struct {
	name, display string
}
type HtmlDropdown struct {
	name    string
	options []option
}

func NewHtmlDropdown(name string) *HtmlDropdown {
	return &HtmlDropdown{name, []option{option{"", ""}}}
}

func (h *HtmlDropdown) Add(name, display string) *HtmlDropdown {
	h.options = append(h.options, option{name, display})
	return h
}

func (h *HtmlDropdown) Template() template.HTML {
	return template.HTML(h.String())
}

func (h *HtmlDropdown) String() string {
	buf := bytes.NewBufferString(u.Format("<select name=\"%s\">\n", h.name))
	for _, o := range h.options {
		buf.WriteString(u.Format("<option value=\"%s\">%s</option>\n", o.name, o.display))
	}
	buf.WriteString("</select>")
	return buf.String()
}
