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

package app

import (
	"bytes"

	"github.com/gin-gonic/gin"
	"github.com/venicegeo/vzutil-versioning/common/table"
	s "github.com/venicegeo/vzutil-versioning/web/app/structs"
	"github.com/venicegeo/vzutil-versioning/web/es"
	u "github.com/venicegeo/vzutil-versioning/web/util"
)

func (a *Application) reportRefOnProject(c *gin.Context) {
	proj := c.Param("proj")
	var form struct {
		Back string `form:"button_back"`
		Ref  string `form:"button_submit"`
	}
	if err := c.Bind(&form); err != nil {
		c.String(400, "Unable to bind form: %s", err.Error())
		return
	}
	if form.Back != "" {
		c.Redirect(303, "/project/"+proj)
		return
	}
	h := gin.H{"report": ""}
	refs, err := a.rtrvr.ListRefsInProj(proj)
	if err != nil {
		h["refs"] = u.Format("Unable to retrieve this projects refs: %s", err.Error())
	} else {
		buttons := s.NewHtmlCollection()
		for _, ref := range refs {
			buttons.Add(s.NewHtmlButton2("button_submit", ref))
			buttons.Add(s.NewHtmlBr())
		}
		h["refs"] = buttons.Template()
	}
	if form.Ref != "" {
		deps, err := a.rtrvr.DepsByRefInProject(proj, form.Ref)
		if err != nil {
			h["report"] = u.Format("Unable to generate report: %s", err.Error())
		} else {
			h["report"] = a.reportAtRefWrk(form.Ref, deps)
		}
	}
	c.HTML(200, "reportref.html", h)
}

func (a *Application) reportAtRefWrk(ref string, deps ReportByRefS) string {
	buf := bytes.NewBufferString("")
	for name, depss := range deps {
		buf.WriteString(u.Format("%s at %s\n%s", name, ref, depss.sha))
		t := table.NewTable(3, len(depss.deps))
		for _, dep := range depss.deps {
			t.Fill(dep.Name, dep.Version, dep.Language)
		}
		buf.WriteString(u.Format("\n%s\n\n", t.NoRowBorders().SpaceColumn(1).Format().String()))
	}
	return buf.String()
}

func (a *Application) reportAtShaWrk(name, sha string, deps []es.Dependency) string {
	buf := bytes.NewBufferString("")
	buf.WriteString(u.Format("%s at %s", name, sha))
	t := table.NewTable(3, len(deps))
	for _, dep := range deps {
		t.Fill(dep.Name, dep.Version, dep.Language)
	}
	buf.WriteString(t.NoRowBorders().SpaceColumn(1).Format().String())
	return buf.String()
}
