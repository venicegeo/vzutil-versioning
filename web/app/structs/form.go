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
	"reflect"
	"strings"
)

type Forms string

var ReportTag Forms = "reporttag"
var ReportSha Forms = "reportsha"
var ListTags Forms = "listtags"
var ListShas Forms = "listshas"
var GenerateTag Forms = "generatetag"
var GenerateSha Forms = "generatesha"
var Differences Forms = "diffman"

type Form struct {
	//Reporting
	ReportTagOrg       string `form:"reporttagorg"`
	ReportTagRepo      string `form:"reporttagrepo"`
	ReportTagTag       string `form:"reporttagtag"`
	ButtonReportAllTag string `form:"button_reporttag"`

	ReportShaOrg       string `form:"reportshaorg"`
	ReportShaRepo      string `form:"reportsharepo"`
	ReportShaSha       string `form:"reportshasha"`
	ButtonReportTagSha string `form:"button_reportsha"`

	//Listing
	TagsOrg        string `form:"tagsorg"`
	TagsRepo       string `form:"tagsrepo"`
	ButtonListTags string `form:"button_listtags"`

	ShasOrg        string `form:"shasorg"`
	ShasRepo       string `form:"shasrepo"`
	ButtonListShas string `form:"button_listshas"`

	//Generation
	AllTagOrg         string `form:"alltagorg"`
	AllTagRepo        string `form:"alltagrepo"`
	ButtonGenerateTag string `form:"button_generatetag"`

	ByShaOrg          string `form:"byshaorg"`
	ByShaRepo         string `form:"bysharepo"`
	ByShaSha          string `form:"byshasha"`
	ButtonGenerateSha string `form:"button_generatesha"`

	//Differences
	ButtonDifferences string `form:"button_diffman"`
}

func (f *Form) IsEmpty() bool {
	val := reflect.ValueOf(f).Elem()
	for i := 0; i < val.NumField(); i++ {
		if strings.TrimSpace(val.Field(i).String()) != "" {
			return false
		}
	}
	return true
}

func (f *Form) FindButtonPress() Forms {
	val := reflect.ValueOf(f).Elem()
	for i := 0; i < val.NumField(); i++ {
		f := val.Field(i)
		field := val.Type().Field(i)
		if strings.HasPrefix(field.Tag.Get("form"), "button_") {
			if f.String() != "" {
				return Forms(strings.TrimPrefix(field.Tag.Get("form"), "button_"))
			}
		}
	}
	return ""
}
