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
	p "github.com/venicegeo/pz-gocommon/gocommon"
	"github.com/venicegeo/vzutil-versioning/common/table"
	s "github.com/venicegeo/vzutil-versioning/web/app/structs/html"
	"github.com/venicegeo/vzutil-versioning/web/es"
	"github.com/venicegeo/vzutil-versioning/web/es/types"
	u "github.com/venicegeo/vzutil-versioning/web/util"
)

func (a *Application) projectsOverview(c *gin.Context) {
	var form struct {
		ProjectId string `form:"button_project"`
		Util      string `form:"button_util"`
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
	case "Custom Compare":
		c.Redirect(303, "/cdiff")
		return
	}
	if form.ProjectId == "" {
		table := s.NewHtmlTable()
		makeButton := func(name string) *s.HtmlSubmitButton {
			return s.NewHtmlSubmitButton3("button_project", name, "button")
		}
		projs, err := a.rtrvr.GetAllProjects()
		if err != nil {
			c.String(500, "Error collecting projects: %s", err.Error())
			return
		}
		projs = append(projs, &Project{a.index, &types.Project{DisplayName: "Add New"}})
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
	} else if form.ProjectId == "Add New" {
		c.Redirect(303, "/newproj")
	} else {
		q := map[string]interface{}{
			"query": es.NewTerm(types.Project_QField_DisplayName, form.ProjectId),
			"size":  1,
		}
		resp, err := a.index.SearchByJSON(Project_QType, q)
		if err != nil {
			c.String(500, "Error getting this project: %s", err.Error())
			return
		}
		if resp.Hits.TotalHits != 1 {
			c.String(400, "This project does not appear to exist")
			return
		}
		var proj types.Project
		if err = json.Unmarshal(*resp.Hits.Hits[0].Source, &proj); err != nil {
			c.String(500, "Unable to unmarshal project: %s", err.Error())
			return
		}
		c.Redirect(303, "/project/"+proj.Id)
	}
}

func (a *Application) newProject(c *gin.Context) {
	type form struct {
		Back        string `form:"button_back"`
		ProjectName string `form:"projectname"`
		Create      string `form:"button_submit"`
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
		displayName := strings.TrimSpace(f.ProjectName)
		id := p.NewUuid().String()
		//TODO query for one
		exists, err := a.index.ItemExists(Project_QType, id)
		if err != nil {
			c.String(500, "Error checking exists in db: %s", err.Error())
			return
		} else if exists {
			c.String(400, "This project already exists")
			return
		}
		if resp, err := a.index.PostDataWait(Project_QType, id, types.NewProject(id, displayName)); err != nil {
			c.String(500, "Error creating project in db: %s", err.Error())
			return
		} else if !resp.Created {
			c.String(500, "Project could not be created for unknown reason")
			return
		}

		c.Redirect(303, "/ui")
	} else {
		c.HTML(200, "newproj.html", nil)
	}
}

func (a *Application) deleteProject(c *gin.Context) {
	var form struct {
		Submit string `form:"button_submit"`
	}
	if err := c.Bind(&form); err != nil {
		c.String(500, "Could not bind form: %s", err.Error())
	}
	projId := c.Param("proj")
	switch form.Submit {
	case "YES":
	case "no":
		c.Redirect(303, "/project/"+projId)
		return
	case "":
		c.HTML(200, "confirmation.html", nil)
		return
	default:
		c.String(400, "Stop trying to break this please")
		return
	}
	if exists, err := a.index.ItemExists(Project_QType, projId); err != nil {
		c.String(500, "Error checking status of project: %s", err.Error())
		return
	} else if !exists {
		c.String(400, "Why would you give me a project that doesnt exist?")
		return
	}
	a.index.DeleteByID(Project_QType, projId)
	if hits, err := es.GetAll(a.index, Repository_QType, es.NewTerm(types.Repository_QField_ProjectId, projId)); err == nil {
		for _, hit := range hits.Hits {
			a.index.DeleteByID(Repository_QType, hit.Id)
		}
	}
	if hits, err := es.GetAll(a.index, RepositoryEntry_QType, es.NewTerm(types.Scan_QField_ProjectId, projId)); err == nil {
		for _, hit := range hits.Hits {
			a.index.DeleteByID(RepositoryEntry_QType, hit.Id)
		}
	}
	c.Redirect(303, "/ui")
}

func (a *Application) reportSha(c *gin.Context) {
	var form struct {
		Back string `form:"button_back"`

		Org  string `form:"org"`
		Repo string `form:"repo"`
		Sha  string `form:"sha"`
		Scan string `form:"button_scan"`

		Files  []string `form:"files[]"`
		Submit string   `form:"button_submit"`
	}
	if err := c.Bind(&form); err != nil {
		c.String(400, "Error binding form: %s", err.Error())
		return
	}
	form.Org = strings.TrimSpace(form.Org)
	form.Repo = strings.TrimSpace(form.Repo)
	form.Sha = strings.TrimSpace(form.Sha)
	if form.Back != "" {
		c.Redirect(303, "/ui")
		return
	}
	h := gin.H{
		"org":        form.Org,
		"repo":       form.Repo,
		"sha":        form.Sha,
		"hidescan":   true,
		"hidereport": true,
	}
	repoName := u.Format("%s/%s", form.Org, form.Repo)
	setScan := func(i interface{}) {
		h["hidescan"] = false
		switch i.(type) {
		case string:
			h["scan"] = s.NewHtmlString(i.(string)).Template()
		case []string:
			check := s.NewHtmlCheckbox("files[]")
			for _, file := range i.([]string) {
				check.Add(file, file, true)
			}
			h["scan"] = s.NewHtmlCollection(check, s.NewHtmlSubmitButton2("button_submit", "Submit")).Template()
		default:
			panic("Youre doing this wrong")
		}
	}
	primaryScan := func() {
		if !a.checkRepoIsReal(form.Org, form.Repo) {
			setScan("This isnt a real repo")
		} else {
			if files, err := a.wrkr.snglRnnr.ScanWithSingle(repoName); err != nil {
				setScan(err.Error())
			} else {
				for i, f := range files {
					files[i] = strings.TrimPrefix(f, repoName)
				}
				setScan(files)
			}
		}
	}
	getReport := func() string {
		ret := make(chan *types.Scan, 2)
		defer func() {
			close(ret)
		}()
		repo := &types.Repository{Fullname: repoName, DependencyInfo: types.RepositoryDependencyInfo{repoName, types.IncomingSha, "", form.Files}}
		a.wrkr.AddTask(&SingleRunnerRequest{&Repository{nil, nil, repo}, form.Sha, ""}, nil, ret)
		scan := <-ret
		if scan == nil {
			return "Generating this sha resulted in an unknown error"
		}
		return a.reportAtShaWrk(scan)
	}
	files := s.NewHtmlCollection()
	for _, f := range form.Files {
		files.Add(s.NewHtmlTextField("files[]", f).Special("readonly"))
	}
	if form.Scan != "" {
		primaryScan()
	} else if form.Submit != "" {
		h["hidescan"] = false
		h["hidereport"] = false
		h["scan"] = files.Template()
		h["report"] = getReport()
	}
	c.HTML(200, "customsha.html", h)
}

func (a *Application) customDiff(c *gin.Context) {
	var form struct {
		Back string `form:"button_back"`

		Org  string `form:"org"`
		Repo string `form:"repo"`
		Scan string `form:"button_scan"`

		Files []string `form:"files[]"`
		Next  string   `form:"button_next"`

		OldSha string `form:"oldsha"`
		NewSha string `form:"newsha"`
		Submit string `form:"button_submit"`
	}
	if err := c.Bind(&form); err != nil {
		c.String(400, "Error binding form: %s", err.Error())
		return
	}
	if form.Back != "" {
		c.Redirect(303, "/ui")
		return
	}
	form.Org = strings.TrimSpace(form.Org)
	form.Repo = strings.TrimSpace(form.Repo)
	form.OldSha = strings.TrimSpace(form.OldSha)
	form.NewSha = strings.TrimSpace(form.NewSha)
	h := gin.H{
		"org":      form.Org,
		"repo":     form.Repo,
		"oldsha":   form.OldSha,
		"newsha":   form.NewSha,
		"hidescan": true,
		"hideshas": true,
		"hidediff": true,
	}
	repoName := u.Format("%s/%s", form.Org, form.Repo)
	setScan := func(i interface{}) {
		h["hidescan"] = false
		switch i.(type) {
		case string:
			h["scan"] = s.NewHtmlString(i.(string)).Template()
		case []string:
			check := s.NewHtmlCheckbox("files[]")
			for _, file := range i.([]string) {
				check.Add(file, file, true)
			}
			h["scan"] = s.NewHtmlCollection(check, s.NewHtmlSubmitButton2("button_next", "Next")).Template()
		default:
			panic("Youre doing this wrong")
		}
	}
	primaryScan := func() {
		if !a.checkRepoIsReal(form.Org, form.Repo) {
			setScan("This isnt a real repo")
		} else {
			if files, err := a.wrkr.snglRnnr.ScanWithSingle(repoName); err != nil {
				setScan(err.Error())
			} else {
				for i, f := range files {
					files[i] = strings.TrimPrefix(f, repoName)
				}
				setScan(files)
			}
		}
	}
	files := s.NewHtmlCollection()
	for i := 0; i < len(form.Files); i++ {
		files.Add(s.NewHtmlTextField("files[]", form.Files[i]).Special("readonly"))
		if i < len(form.Files)-1 {
			files.Add(s.NewHtmlBr())
		}
	}
	if form.Scan != "" {
		primaryScan()
	} else if form.Next != "" {
		h["hidescan"] = false
		h["hideshas"] = false
		h["scan"] = files.Template()
	} else if form.Submit != "" {
		h["hidescan"] = false
		h["hideshas"] = false
		h["scan"] = files.Template()
		h["hidediff"] = false
		diff, err := a.diffMan.ShaCompare(repoName, form.Files, form.OldSha, form.NewSha)
		if err != nil {
			h["diff"] = err.Error()
		} else if diff == nil {
			h["diff"] = "These are identical"
		} else {
			height := len(diff.Added)
			if height < len(diff.Removed) {
				height = len(diff.Removed)
			}
			t := table.NewTable(2, height+1).NoRowBorders()
			t.Fill("Removed", "Added")
			for i := 0; i < height; i++ {
				if i >= len(diff.Removed) {
					t.Fill("")
				} else {
					t.Fill(diff.Removed[i])
				}
				if i >= len(diff.Added) {
					t.Fill("")
				} else {
					t.Fill(diff.Added[i])
				}
			}
			h["diff"] = t.HasHeading().Format().String()
		}
	}
	c.HTML(200, "customdiff.html", h)
}
