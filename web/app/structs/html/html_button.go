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
	"html/template"

	f "github.com/venicegeo/vzutil-versioning/web/util"
)

type HtmlButton struct {
	display string
	name    string
	value   string
	typ     string
	style   string
}

func NewHtmlButton(display, name, value, typ string) *HtmlButton {
	return &HtmlButton{display, name, value, typ, ""}
}
func (h *HtmlButton) Style(style string) *HtmlButton {
	h.style = style
	return h
}

func (h *HtmlButton) Template() template.HTML {
	return template.HTML(h.String())
}

func (h *HtmlButton) String() string {
	return f.Format(`<button name="%s" value="%s" type="%s" style="%s">%s</button>`, h.name, h.value, h.typ, h.style, h.display)
}
