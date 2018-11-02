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

type HtmlTextField struct {
	name, value, special string
}

func NewHtmlTextField(name, value string) *HtmlTextField {
	return &HtmlTextField{name, value, ""}
}

func (h *HtmlTextField) Special(special string) *HtmlTextField {
	h.special = special
	return h
}

func (h *HtmlTextField) Template() template.HTML {
	return template.HTML(h.String())
}

func (h *HtmlTextField) String() string {
	return f.Format(`<input type="text" name="%s" value="%s" %s>`, h.name, h.value, h.special)
}
