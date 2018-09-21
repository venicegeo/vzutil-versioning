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
	"strings"

	f "github.com/venicegeo/vzutil-versioning/web/util"
)

type HtmlSubmitButton struct {
	name    string
	value   string
	class   string
	special string
}

func NewHtmlSubmitButton(value string) *HtmlSubmitButton {
	return &HtmlSubmitButton{"button_" + strings.ToLower(value), value, "", ""}
}
func NewHtmlSubmitButton2(name, value string) *HtmlSubmitButton {
	return &HtmlSubmitButton{name, value, "", ""}
}
func NewHtmlSubmitButton3(name, value, class string) *HtmlSubmitButton {
	return &HtmlSubmitButton{name, value, class, ""}
}

func (h *HtmlSubmitButton) Special(special string) *HtmlSubmitButton {
	h.special = special
	return h
}

func (h *HtmlSubmitButton) Template() template.HTML {
	return template.HTML(h.String())
}

func (h *HtmlSubmitButton) String() string {
	return f.Format(`<input type="submit" name="%s" value="%s" class="%s" %s>`, h.name, h.value, h.class, h.special)
}
