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
	u "github.com/venicegeo/vzutil-versioning/web/util"
)

func (a *Application) listShas(c *gin.Context) {
	if a.checkBack(c) {
		return
	}
	fullName := u.Format("%s/%s", c.Param("org"), c.Param("repo"))
	refShas, count, err := a.rtrvr.ListShas(fullName)
	if err != nil {
		a.displayFailure(c, err.Error())
		return
	}
	buf := bytes.NewBufferString(u.Format("List of Shas for %s\n", fullName))
	t := table.NewTable(2, count+len(refShas))
	for refName, shas := range refShas {
		t.Fill(refName, "")
		for _, sha := range shas {
			t.Fill("", sha)
		}
	}
	buf.WriteString(t.NoRowBorders().Format().String())
	a.displaySuccess(c, buf.String())
}

func (a *Application) listRefsRepo(c *gin.Context) {
	if a.checkBack(c) {
		return
	}
	fullName := u.Format("%s/%s", c.Param("org"), c.Param("repo"))
	refs, err := a.rtrvr.ListRefsRepo(fullName)
	if err != nil {
		a.displayFailure(c, err.Error())
		return
	}
	buf := bytes.NewBufferString(u.Format("List of refs for %s\n", fullName))
	t := table.NewTable(1, len(refs))
	for _, r := range refs {
		t.Fill(r)
	}
	buf.WriteString(t.NoRowBorders().NoColumnBorders().Format().String())
	a.displaySuccess(c, buf.String())
}

func (a *Application) listRefs(c *gin.Context) {
	if a.checkBack(c) {
		return
	}
	org := c.Param("org")
	tags, num, err := a.rtrvr.ListRefsInProjByRepo(org)
	if err != nil {
		a.displayFailure(c, err.Error())
		return
	}
	buf := bytes.NewBufferString(u.Format("List of refs for %s\n", org))
	t := table.NewTable(2, num+len(*tags))
	for k, v := range *tags {
		if len(v) == 0 {
			continue
		}
		t.Fill("")
		t.Fill("")
		t.Fill(k)
		for i, vv := range v {
			t.Fill(vv)
			if i != len(v)-1 {
				t.Fill(" ")
			}
		}
	}
	buf.WriteString(t.SpaceColumn(1).NoRowBorders().NoColumnBorders().Format().String())
	a.displaySuccess(c, buf.String())
}

func (a *Application) listRepositories(c *gin.Context) {
	if a.checkBack(c) {
		return
	}
	ps, err := a.rtrvr.ListRepositories()
	a.listRepositoriesWrk(ps, err, bytes.NewBufferString("List of repositories\n"), c)
}
func (a *Application) listRepositoriesProj(c *gin.Context) {
	if a.checkBack(c) {
		return
	}
	proj := c.Param("proj")
	ps, err := a.rtrvr.ListRepositoriesByProj(proj)
	a.listRepositoriesWrk(ps, err, bytes.NewBufferString(u.Format("List of repositories for %s\n", proj)), c)
}
func (a *Application) listRepositoriesWrk(ps []string, err error, buf *bytes.Buffer, c *gin.Context) {
	if err != nil {
		a.displayFailure(c, err.Error())
		return
	}
	t := table.NewTable(1, len(ps))
	for _, v := range ps {
		t.Fill(v)
	}
	buf.WriteString(t.NoRowBorders().NoColumnBorders().Format().String())
	a.displaySuccess(c, buf.String())
}
