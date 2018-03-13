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

const DepSearch Forms = "depsearch"
const ReportRef Forms = "reportref"
const ReportSha Forms = "reportsha"
const ListRefs Forms = "listrefs"
const ListShas Forms = "listshas"
const GenerateTag Forms = "generatetag"
const GenerateBranch Forms = "generatebranch"
const Differences Forms = "diffman"
const CustomDifference Forms = "customdiff"

type Form struct {
	ButtonDepSearch string `form:"button_depsearch"`

	//Reporting
	ReportRefOrg       string `form:"reportreforg"`
	ReportRefRepo      string `form:"reportrefrepo"`
	ReportRefRef       string `form:"reportreftag"`
	ButtonReportAllRef string `form:"button_reportref"`

	ReportShaOrg       string `form:"reportshaorg"`
	ReportShaRepo      string `form:"reportsharepo"`
	ReportShaSha       string `form:"reportshasha"`
	ButtonReportTagSha string `form:"button_reportsha"`

	//Listing
	RefsOrg        string `form:"refsorg"`
	RefsRepo       string `form:"refsrepo"`
	ButtonListRefs string `form:"button_listrefs"`

	ShasOrg        string `form:"shasorg"`
	ShasRepo       string `form:"shasrepo"`
	ButtonListShas string `form:"button_listshas"`

	//Generation
	AllTagOrg         string `form:"alltagorg"`
	AllTagRepo        string `form:"alltagrepo"`
	ButtonGenerateTag string `form:"button_generatetag"`

	BranchOrg            string `form:"branchorg"`
	BranchRepo           string `form:"branchrepo"`
	BranchBranch         string `form:"branchbranch"`
	ButtonGenerateBranch string `form:"button_generatebranch"`

	//Differences
	ButtonDifferences      string `form:"button_diffman"`
	ButtonCustomDifference string `form:"button_customdiff"`
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
