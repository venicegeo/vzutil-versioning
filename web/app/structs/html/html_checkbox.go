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
)

type checkboxinfo struct {
	value, text string
	selected    bool
}
type HtmlCheckbox struct {
	name string
	info []checkboxinfo
}

func NewHtmlCheckbox(name string) *HtmlCheckbox {
	return &HtmlCheckbox{name, []checkboxinfo{}}
}

func (h *HtmlCheckbox) Add(value, text string, selected bool) *HtmlCheckbox {
	h.info = append(h.info, checkboxinfo{value, text, selected})
	return h
}
func (h *HtmlCheckbox) Template() template.HTML {
	return template.HTML(h.String())
}
func (h *HtmlCheckbox) String() string {
	buf := bytes.NewBufferString("")
	for i := 0; i < len(h.info); i++ {
		buf.WriteString(fmt.Sprintf(`<input type="checkbox" name="%s" value="%s" `, h.name, h.info[i].value))
		if h.info[i].selected {
			buf.WriteString("checked")
		}
		buf.WriteString(fmt.Sprintf(" >%s<br>", h.info[i].text))
	}
	return buf.String()
}
