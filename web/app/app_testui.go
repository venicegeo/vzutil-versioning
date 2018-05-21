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
	"encoding/json"
	"strings"

	"github.com/gin-gonic/gin"
	nt "github.com/venicegeo/pz-gocommon/gocommon"
	"github.com/venicegeo/vzutil-versioning/common/table"
	s "github.com/venicegeo/vzutil-versioning/web/app/structs"
	"github.com/venicegeo/vzutil-versioning/web/es"
	u "github.com/venicegeo/vzutil-versioning/web/util"
)

func (a *Application) test(c *gin.Context) {
	type form struct {
		Project string `form:"button_project"`
	}
	var f form
	_ = c.Bind(&f)
	if f.Project == "" {
		table := s.NewHtmlTable()
		makeButton := func(name string) *s.HtmlButton {
			return s.NewHtmlButton3("button_project", name, "button")
		}
		projs, err := a.rtrvr.ListProjects()
		if err != nil {
			c.String(500, "Error collecting projects: %s", err.Error())
			return
		}
		projs = append(projs, &es.Project{DisplayName: "Add New"})
		row := -1
		for i, proj := range projs {
			if i%3 == 0 {
				table.AddRow()
				row++
			}
			table.AddItem(row, makeButton(proj.DisplayName))
		}
		h := gin.H{}
		h["table"] = table.Template()
		c.HTML(200, "test.html", h)
		return
	} else if f.Project == "Add New" {
		c.Redirect(303, "/newproj")
	} else {
		resp, err := a.index.SearchByJSON("project", u.Format(`{
	"query":{
		"term":{
			"displayname":"%s"
		}
	},
	"size":1
}`, f.Project))
		if err != nil {
			c.String(500, "Error getting this project: %s", err.Error())
			return
		}
		if resp.NumHits() != 1 {
			c.String(400, "This project does not appear to exist")
			return
		}
		var proj es.ProjectEntry
		if err = json.Unmarshal(*resp.GetHit(0).Source, &proj); err != nil {
			c.String(500, "Unable to unmarshal project: %s", err.Error())
			return
		}
		c.Redirect(303, "/project/"+proj.Name)
	}
}

func (a *Application) newProj(c *gin.Context) {
	type form struct {
		Back        string   `form:"button_back"`
		ProjectName string   `form:"projectname"`
		Repos       []string `form:"repos[]"`
		Checkouts   []string `form:"checkout[]"`
		Create      string   `form:"button_submit"`
	}
	var f form
	if err := c.Bind(&f); err != nil {
		c.String(500, "Unable to bind form: %s", err.Error())
		return
	}
	if f.Back != "" {
		c.Redirect(303, "/test")
	} else if f.Create != "" {
		if f.ProjectName == "" {
			c.String(400, "You must specify a project name")
			return
		}
		realr := []string{}
		for _, r := range f.Repos {
			if strings.TrimSpace(r) == "" {
				continue
			}
			realr = append(realr, strings.TrimSpace(strings.ToLower(r)))
		}
		displayName := f.ProjectName
		name := strings.ToLower(strings.Replace(strings.Replace(f.ProjectName, "/", "_", -1), " ", "", -1))
		exists, err := a.index.ItemExists("project", name)
		if err != nil {
			c.String(500, "Error checking exists in db: %s", err.Error())
			return
		} else if exists {
			c.String(400, "This project already exists")
			return
		}
		if resp, err := a.index.PostData("project", name, es.Project{name, displayName}); err != nil {
			c.String(500, "Error creating project in db: %s", err.Error())
			return
		} else if !resp.Created {
			c.String(500, "Project could not be created for unknown reason")
			return
		}

		//TODO thread
		for _, fullName := range realr {
			checkRepoExist := func(fullName string) error {
				url := u.Format("https://github.com/%s", fullName)
				if code, _, _, e := nt.HTTP(nt.HEAD, url, nt.NewHeaderBuilder().GetHeader(), nil); e != nil {
					return e
				} else if code != 200 {
					return u.Error("Code on %s is no 200, but %d", fullName, code)
				} else {
					return nil
				}
			}
			err := checkRepoExist(fullName)
			if err != nil {
				continue
			}
			a.index.PostData("project_entry", "", es.ProjectEntry{name, fullName})
		}

		c.Redirect(303, "/test")
	} else {
		c.HTML(200, "newproj.html", nil)
	}
}

func (a *Application) testProject(c *gin.Context) {
	proj := c.Param("proj")
	var f struct {
		Util string `form:"button_util"`
		Sha  string `form:"button_sha"`
		Gen  string `form:"button_gen"`
	}
	if err := c.Bind(&f); err != nil {
		c.String(400, "Unable to bind form: %s", err.Error())
		return
	}
	depsStr := ""
	if f.Util != "" {
		switch f.Util {
		case "Report By Ref":
			println("report")
			return
		case "Generate All Tags":
			println("generate")
			return
		case "Add Repository":
			println("add")
			return
		case "Remove Repository":
			println("remove")
			return
		}
	} else if f.Sha != "" {
		deps, fullName, _, found, err := a.rtrvr.DepsBySha(f.Sha)
		if !found && err != nil {
			c.String(400, "Unable to find this sha: %s", err.Error())
			return
		} else if found && err != nil {
			c.String(500, "Unable to obtain the results: %s", err.Error())
			return
		}
		hDeps := bytes.NewBufferString("")
		hDeps.WriteString("Report for ")
		hDeps.WriteString(fullName)
		hDeps.WriteString(" at ")
		hDeps.WriteString(f.Sha)
		hDeps.WriteString("\n")
		t := table.NewTable(3, len(deps))
		for _, dep := range deps {
			t.Fill(dep.Name, dep.Version, dep.Language)
		}
		hDeps.WriteString(t.SpaceColumn(1).NoRowBorders().Format().String())
		depsStr = hDeps.String()
	} else if f.Gen != "" {
		repoFullName := strings.TrimPrefix(f.Gen, "Generate Branch - ")
		println(repoFullName)
		return
	}
	accord := s.NewHtmlAccordion()
	repos, err := a.rtrvr.ListRepositoriesByProj(proj)
	if err != nil {
		c.String(500, "Unable to retrieve repository list: %s", err.Error())
		return
	}
	for _, repoName := range repos {
		refs, err := a.rtrvr.ListRefsRepo(repoName)
		if err != nil {
			c.String(500, "Unable to retrieve refs:  %s", err.Error())
			return
		}
		tempAccord := s.NewHtmlAccordion()
		shas, _, err := a.rtrvr.ListShas(repoName)
		if err != nil {
			c.String(500, "Unable to retrieve shas: %s", err.Error())
			return
		}
		for _, ref := range refs {
			c := s.NewHtmlCollection()
			for _, sha := range shas["refs/"+ref] {
				c.Add(s.NewHtmlButton2("button_sha", sha))
			}
			tempAccord.AddItem(ref, s.NewHtmlForm(c))
		}
		accord.AddItem(repoName, s.NewHtmlCollection(s.NewHtmlForm(s.NewHtmlButton2("button_gen", "Generate Branch - "+repoName)), tempAccord.Sort()))
	}
	accord.Sort()
	h := gin.H{}
	h["accordion"] = accord.Template()
	h["deps"] = depsStr
	c.HTML(200, "test2.html", h)
}

func (a *Application) addRepo(c *gin.Context) {
	proj := c.Param("proj")
	currentRepos, err := a.rtrvr.ListRepositoriesByProj(proj)
	if err != nil {
		c.String(400, "Error getting the projects repositories: %s", err.Error())
		return
	}
	c.String(200, "%s", currentRepos)
}

func (a *Application) genBranch(c *gin.Context) {
	var form struct {
		Back string `form:"button_back"`
		Gen  string `form:"button_generatebranch"`
	}
	pproj := c.Param("proj")
	porg := c.Param("org")
	prepo := c.Param("repo")
	if form.Back != "" {
		c.Redirect(303, "/test/"+pproj)
		return
	}
	if form.Gen != "" {
	}
}
