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
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	nt "github.com/venicegeo/pz-gocommon/gocommon"
	"github.com/venicegeo/vzutil-versioning/common/table"
	s "github.com/venicegeo/vzutil-versioning/web/app/structs"
	"github.com/venicegeo/vzutil-versioning/web/es"
	u "github.com/venicegeo/vzutil-versioning/web/util"
)

func (a *Application) reportSha(c *gin.Context) {
	if a.checkBack(c) {
		return
	}
	fullName := u.Format("%s/%s", c.Param("org"), c.Param("repo"))
	sha := c.Param("sha")
	deps, found, err := a.rprtr.ReportByShaName(fullName, sha)
	if err != nil || !found {
		{
			code, _, _, err := nt.HTTP(nt.HEAD, u.Format("https://github.com/%s/commit/%s", fullName, sha), nt.NewHeaderBuilder().GetHeader(), nil)
			if err != nil {
				c.String(400, "Could not verify this sha: "+err.Error())
				return
			}
			if code != 200 {
				c.String(400, u.Format("Could not verify this sha, head code: %d", code))
				return
			}
		}
		res := a.wrkr.CloneWork(&s.GitWebhook{AfterSha: sha, Repository: s.GitRepository{FullName: fullName}})
		if res == nil {
			c.String(400, "Sha [%s] did not previously exist and could not be generated", sha)
			return
		}
		deps = res.Deps
		sort.Sort(es.DependencySort(deps))
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

func (a *Application) reportRef(c *gin.Context) {
	if a.checkBack(c) {
		return
	}
	reforg := c.Param("reforg")
	refrepo := c.Param("refrepo")
	ref := c.Param("ref")

	var deps map[string][]es.Dependency
	var err error
	if reforg != "" && refrepo != "" && ref != "" {
		ref = strings.Replace(ref, "_", `/`, -1)
		deps, err = a.rprtr.ReportByRef(reforg, refrepo, ref)
	} else if reforg != "" && refrepo != "" {
		ref = strings.Replace(refrepo, "_", `/`, -1)
		deps, err = a.rprtr.ReportByRef(reforg, ref)
	} else if reforg != "" {
		ref = strings.Replace(reforg, "_", `/`, -1)
		deps, err = a.rprtr.ReportByRef(ref)
	}

	if err != nil {
		a.displayFailure(c, "Unable to do this: "+err.Error())
		return
	}
	res := ""
	for name, depss := range deps {
		res += name + " at " + ref
		t := table.NewTable(3, len(depss))
		for _, dep := range depss {
			t.Fill(dep.Name)
			t.Fill(dep.Version)
			t.Fill(dep.Language)
		}
		res += "\n" + t.NoRowBorders().SpaceColumn(1).Format().String() + "\n\n"
	}
	a.displaySuccess(c, res)
}
