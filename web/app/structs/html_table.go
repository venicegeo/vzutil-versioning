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
)

type HtmlTable struct {
	table [][]string
}

func NewHtmlTable() *HtmlTable {
	return &HtmlTable{[][]string{}}
}

func (h *HtmlTable) AddRow() *HtmlTable {
	h.table = append(h.table, []string{})
	return h
}

func (h *HtmlTable) AddItem(row int, elem string) *HtmlTable {
	h.table[row] = append(h.table[row], elem)
	return h
}

func (h *HtmlTable) Template() template.HTML {
	return template.HTML(h.String())
}

func (h *HtmlTable) String() string {
	res := "<table>\n"
	for _, row := range h.table {
		res += "\t<tr>\n"
		for _, item := range row {
			res += "\t\t<td>" + item + "</td>\n"
		}
		res += "\t</tr>\n"
	}
	res += "</table>"
	return res
}
