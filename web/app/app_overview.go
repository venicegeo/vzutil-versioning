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
	"encoding/json"
	"strings"

	"github.com/gin-gonic/gin"
	s "github.com/venicegeo/vzutil-versioning/web/app/structs"
	"github.com/venicegeo/vzutil-versioning/web/es"
	u "github.com/venicegeo/vzutil-versioning/web/util"
)

func (a *Application) projectsOverview(c *gin.Context) {
	var form struct {
		Project string `form:"button_project"`
		Util    string `form:"button_util"`
	}
	if err := c.Bind(&form); err != nil {
		c.String(400, "Error binding the form: %s", err.Error())
		return
	}
	switch form.Util {
	case "Report By Sha":
		c.Redirect(303, "/reportsha")
		return
	case "Dependency Search":
		c.Redirect(303, "/depsearch")
		return
	}
	if form.Project == "" {
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
		c.HTML(200, "overview.html", h)
		return
	} else if form.Project == "Add New" {
		c.Redirect(303, "/newproj")
	} else {
		resp, err := a.index.SearchByJSON("project", u.Format(`{
	"query":{
		"term":{
			"displayname":"%s"
		}
	},
	"size":1
}`, form.Project))
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

func (a *Application) newProject(c *gin.Context) {
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
		c.Redirect(303, "/ui")
	} else if f.Create != "" {
		if f.ProjectName == "" {
			c.String(400, "You must specify a project name")
			return
		}
		if f.ProjectName == "Add New" {
			c.String(400, "You can not do this")
			return
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

		a.addReposToProjWrk(name, f.Repos)

		c.Redirect(303, "/ui")
	} else {
		c.HTML(200, "newproj.html", nil)
	}
}

func (a *Application) reportSha(c *gin.Context) {
	var form struct {
		Back   string `form:"button_back"`
		Org    string `form:"org"`
		Repo   string `form:"repo"`
		Sha    string `form:"sha"`
		Submit string `form:"button_submit"`
	}
	if err := c.Bind(&form); err != nil {
		c.String(400, "Could not bind the form: %s", err.Error())
		return
	}
	if form.Back != "" {
		c.Redirect(303, "/ui")
		return
	}
	h := gin.H{"report": "Report will appear here"}
	if form.Submit != "" {
		fullName := u.Format("%s/%s", form.Org, form.Repo)
		deps, err := a.rtrvr.DepsByShaNameGen(fullName, form.Sha)
		if err != nil {
			h["report"] = u.Format("Unable to run against %s at %s:\n%s", fullName, form.Sha, err.Error())
		} else {
			h["report"] = a.reportAtShaWrk(fullName, form.Sha, deps)
		}
	}
	c.HTML(200, "reportsha.html", h)
}
