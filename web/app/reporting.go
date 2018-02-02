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
	"log"

	"github.com/gin-gonic/gin"
	"github.com/venicegeo/vzutil-versioning/web/es"
	u "github.com/venicegeo/vzutil-versioning/web/util"
	"github.com/venicegeo/vzutil-versioning/web/util/table"
)

func (a *Application) reportSha(c *gin.Context) {
	if a.checkBack(c) {
		return
	}
	fullName := u.Format("%s/%s", c.Param("org"), c.Param("repo"))
	sha := c.Param("sha")
	deps, err := a.rprtr.ReportByShaName(fullName, sha)
	if err != nil {
		c.String(400, "Unable to do this: %s", err.Error())
		return
	}
	header := "Report for " + fullName + " at " + sha + "\n"
	t := table.NewTable(3, len(deps))
	for _, dep := range deps {
		t.Fill(dep.Name)
		t.Fill(dep.Version)
		t.Fill(dep.Language)
	}
	a.displaySuccess(c, header+t.SpaceColumn(1).NoRowBorders().Format().String())
}

func (a *Application) reportTag(c *gin.Context) {
	if a.checkBack(c) {
		return
	}
	tagorg := c.Param("tagorg")
	tagrepo := c.Param("tagrepo")
	tag := c.Param("tag")

	var deps map[string][]es.Dependency
	var err error
	if tagorg != "" && tagrepo != "" && tag != "" {
		deps, err = a.rprtr.ReportByTag(tagorg, tagrepo, tag)
	} else if tagorg != "" && tagrepo != "" {
		tag = tagrepo
		deps, err = a.rprtr.ReportByTag(tagorg, tag)
	} else if tagorg != "" {
		tag = tagorg
		deps, err = a.rprtr.ReportByTag(tag)
	}

	if err != nil {
		a.displayFailure(c, "Unable to do this: "+err.Error())
		return
	}
	res := ""
	for name, depss := range deps {
		res += name + " at " + tag
		t := table.NewTable(3, len(depss))
		for _, dep := range depss {
			log.Println(dep)
			t.Fill(dep.Name)
			t.Fill(dep.Version)
			t.Fill(dep.Language)
		}
		res += "\n" + t.NoRowBorders().SpaceColumn(1).Format().String() + "\n\n"
	}
	a.displaySuccess(c, res)
}
