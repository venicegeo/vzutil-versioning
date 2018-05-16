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
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/venicegeo/vzutil-versioning/common/table"
	"github.com/venicegeo/vzutil-versioning/web/es"
	u "github.com/venicegeo/vzutil-versioning/web/util"
)

func (a *Application) reportSha(c *gin.Context) {
	if a.checkBack(c) {
		return
	}
	sha := c.Param("sha")
	var deps []es.Dependency
	var err error
	found := true
	fullName := "unknown"
	if sha == "" {
		sha = c.Param("shaorg")
		deps, found, err = a.rtrvr.DepsBySha(sha)
	} else {
		fullName = u.Format("%s/%s", c.Param("shaorg"), c.Param("repo"))
		deps, err = a.rtrvr.DepsByShaNameGen(fullName, sha)
	}
	if err != nil {
		c.String(400, err.Error())
		return
	}
	if !found {
		c.String(400, "This sha was not found and the information to generate it was not provided.")
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
		deps, err = a.rtrvr.DepsByRef(reforg, refrepo, ref)
	} else if reforg != "" && refrepo != "" {
		ref = strings.Replace(refrepo, "_", `/`, -1)
		deps, err = a.rtrvr.DepsByRef(reforg, ref)
	} else if reforg != "" {
		ref = strings.Replace(reforg, "_", `/`, -1)
		deps, err = a.rtrvr.DepsByRef(ref)
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
