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
	"log"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	nt "github.com/venicegeo/pz-gocommon/gocommon"
	"github.com/venicegeo/vzutil-versioning/common/table"
	h "github.com/venicegeo/vzutil-versioning/web/app/helpers"
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

func (a *Application) addReposToProjWrk(name string, reposs []string) {
	realr := []string{}
	for _, r := range reposs {
		if strings.TrimSpace(r) == "" {
			continue
		}
		realr = append(realr, strings.TrimSpace(strings.ToLower(r)))
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
		if err := checkRepoExist(fullName); err != nil {
			continue
		}
		a.index.PostData("project_entry", "", es.ProjectEntry{name, fullName})
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

		c.Redirect(303, "/test")
	} else {
		c.HTML(200, "newproj.html", nil)
	}
}

func (a *Application) testProject(c *gin.Context) {
	proj := c.Param("proj")
	var form struct {
		Back string `form:"button_back"`
		Util string `form:"button_util"`
		Sha  string `form:"button_sha"`
		Gen  string `form:"button_gen"`
		Diff string `form:"button_diff"`
	}
	if err := c.Bind(&form); err != nil {
		c.String(400, "Unable to bind form: %s", err.Error())
		return
	}
	if form.Back != "" {
		c.Redirect(303, "/test")
		return
	}
	depsStr := "Result info will appear here"
	if form.Util != "" {
		switch form.Util {
		case "Report By Ref":
			c.Redirect(303, "/reportref/"+proj)
			return
		case "Generate All Tags":
			str, err := a.genTagsWrk(proj)
			if err != nil {
				u.Format("Unable to generate all tags: %s", err.Error())
			} else {
				depsStr = str
			}
		case "Add Repository":
			c.Redirect(303, "/addrepo/"+proj)
			return
		case "Remove Repository":
			c.Redirect(303, "/removerepo/"+proj)
			return
		case "Dependency Search":
			c.Redirect(303, "/depsearch/"+proj)
			return
		}
	} else if form.Sha != "" {
		deps, fullName, _, found, err := a.rtrvr.DepsBySha(form.Sha)
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
		hDeps.WriteString(form.Sha)
		hDeps.WriteString("\n")
		t := table.NewTable(3, len(deps))
		for _, dep := range deps {
			t.Fill(dep.Name, dep.Version, dep.Language)
		}
		hDeps.WriteString(t.SpaceColumn(1).NoRowBorders().Format().String())
		depsStr = hDeps.String()
	} else if form.Gen != "" {
		repoFullName := strings.TrimPrefix(form.Gen, "Generate Branch - ")
		c.Redirect(303, u.Format("/genbranch/%s/%s", proj, repoFullName))
		return
	} else if form.Diff != "" {
		c.Redirect(303, "/diff/"+proj)
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
	{
		diffs, err := a.diffMan.DiffList()
		if err != nil {
			h["diff"] = ""
		} else {
			h["diff"] = u.Format(" (%d)", len(diffs))
		}
	}
	c.HTML(200, "project.html", h)
}

func (a *Application) addRepo(c *gin.Context) {
	var form struct {
		Back   string   `form:"button_back"`
		Repos  []string `form:"repos[]"`
		Create string   `form:"button_submit"`
	}
	proj := c.Param("proj")
	if err := c.Bind(&form); err != nil {
		c.String(400, "Error binding form: %s", err.Error())
		return
	}
	if form.Back != "" {
		c.Redirect(303, "/project/"+proj)
		return
	}
	if form.Create != "" {
		a.addReposToProjWrk(proj, form.Repos)
		c.Redirect(303, "/project/"+proj)
		return
	}
	currentRepos, err := a.rtrvr.ListRepositoriesByProj(proj)
	if err != nil {
		c.String(400, "Error getting the projects repositories: %s", err.Error())
		return
	}
	buf := bytes.NewBufferString("")
	for _, repo := range currentRepos {
		buf.WriteString(repo)
		buf.WriteString("\n")
	}
	h := gin.H{}
	h["current"] = buf.String()
	c.HTML(200, "addrepo.html", h)
}

func (a *Application) genBranch(c *gin.Context) {
	var form struct {
		Back   string `form:"button_back"`
		Gen    string `form:"button_generatebranch"`
		Branch string `form:"branch"`
	}
	if err := c.Bind(&form); err != nil {
		c.String(400, "Could not bind form: %s", err.Error())
	}
	pproj := c.Param("proj")
	porg := c.Param("org")
	prepo := c.Param("repo")
	branch := form.Branch
	if form.Back != "" {
		c.Redirect(303, "/project/"+pproj)
		return
	}
	if form.Gen != "" {
		_, err := a.generateBranchWrk(prepo, u.Format("%s/%s", porg, prepo), branch)
		if err != nil {
			c.String(400, "Could not generate this sha: %s", err.Error())
			return
		}
		c.Redirect(303, "/project/"+pproj)
		return
	}
	h := gin.H{}
	h["org"] = porg
	h["repo"] = prepo
	c.HTML(200, "genbranch.html", h)
}

func (a *Application) reportRefNew(c *gin.Context) {
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
		deps, err := a.rtrvr.DepsByRef(proj, form.Ref)
		if err != nil {
			h["report"] = u.Format("Unable to generate report: %s", err.Error())
		} else {
			buf := bytes.NewBufferString("")
			for name, depss := range deps {
				buf.WriteString(u.Format("%s at %s", name, form.Ref))
				t := table.NewTable(3, len(depss))
				for _, dep := range depss {
					t.Fill(dep.Name)
					t.Fill(dep.Version)
					t.Fill(dep.Language)
				}
				buf.WriteString(u.Format("\n%s\n\n", t.NoRowBorders().SpaceColumn(1).Format().String()))
			}
			h["report"] = buf.String()
		}
	}

	c.HTML(200, "reportref.html", h)
}
func (a *Application) removeRepo(c *gin.Context) {
	proj := c.Param("proj")
	var form struct {
		Back string `form:"button_back"`
		Repo string `form:"button_submit"`
	}
	if err := c.Bind(&form); err != nil {
		c.String(400, "Unable to bind form: %s", err.Error())
		return
	}
	if form.Back != "" {
		c.Redirect(303, "/project/"+proj)
		return
	}
	if form.Repo != "" {
		resp, err := a.index.SearchByJSON("project_entry", u.Format(`{
	"query":{
		"bool":{
			"must":[
				{
					"term":{"name":"%s"}
				},{
					"term":{"repo":"%s"}
				}
			]
		}
	},
	"size":1
}`, proj, form.Repo))
		if err != nil {
			c.String(400, "Unable to find the project entry: %s", err.Error())
			return
		}
		if resp.NumHits() != 1 {
			c.String(400, "Could not find the project entry")
			return
		}
		_, err = a.index.DeleteByIDWait("project_entry", resp.GetHit(0).ID)
		if err != nil {
			c.String(500, "Unable to delete project entry: %s", err.Error())
			return
		}
		c.Redirect(303, "/removerepo/"+proj)
		return
	}
	repos, err := a.rtrvr.ListRepositoriesByProj(proj)
	if err != nil {
		c.String(500, "Unable to get the repos: %s", err)
		return
	}
	h := gin.H{}
	buttons := s.NewHtmlCollection()
	for _, repo := range repos {
		buttons.Add(s.NewHtmlButton2("button_submit", repo))
		buttons.Add(s.NewHtmlBr())
	}
	h["repos"] = buttons.Template()
	c.HTML(200, "removerepo.html", h)
}
func (a *Application) genTagsWrk(proj string) (string, error) {
	repos, err := a.rtrvr.ListRepositoriesByProj(proj)
	if err != nil {
		return "", err
	}
	go func(repos []string) {
		for _, repo := range repos {
			name := strings.SplitN(repo, "/", 2)[1]
			dat, err := h.NewTagsRunner(name, repo).Run()
			if err != nil {
				log.Println("[TAG UPDATER] Was unable to run tags against " + repo + ": [" + err.Error() + "]")
				continue
			}
			go func(dat map[string]string, name string, repo string) {
				for sha, ref := range dat {
					git := s.GitWebhook{
						Ref:      ref,
						AfterSha: sha,
						Repository: s.GitRepository{
							Name:     name,
							FullName: repo,
						},
						Timestamp: time.Now().UnixNano(),
					}
					log.Println(repo, sha, ref)
					a.wbhkRnnr.RunAgainstWeb(&git)
				}
			}(dat, name, repo)
		}
	}(repos)

	buf := bytes.NewBufferString("Trying to run against:\n")
	for _, repo := range repos {
		buf.WriteString("\n")
		buf.WriteString(repo)
	}
	return buf.String(), nil
}
func (a *Application) depSearch(c *gin.Context) {
	proj := c.Param("proj")
	var form struct {
		Back         string `form:"button_back"`
		DepName      string `form:"depsearchname"`
		DepVersion   string `form:"depsearchversion"`
		ButtonSearch string `form:"button_depsearch"`
	}
	if err := c.Bind(&form); err != nil {
		c.String(400, "Unable to bind form: %s", err.Error())
		return
	}
	h := gin.H{
		"data":             "Search Results will appear here",
		"depsearchname":    form.DepName,
		"depsearchversion": form.DepVersion,
	}
	if form.Back != "" {
		c.Redirect(303, "/project/"+proj)
	} else if form.ButtonSearch != "" {
		code, dat := a.searchForDepWrk(form.DepName, form.DepVersion)
		h["data"] = dat
		c.HTML(code, "depsearch.html", h)
	} else {
		c.HTML(200, "depsearch.html", h)
	}
}

func (a *Application) diff(c *gin.Context) {
	proj := c.Param("proj")
	var back struct {
		Back string `form:"button_back"`
	}
	if err := c.Bind(&back); err != nil {
		c.String(400, "Unable to bind form: %s", err.Error())
		return
	}
	if back.Back != "" {
		c.Redirect(303, "/project/"+proj)
		return
	}
	gh := gin.H{}
	gh["buttons"] = "Differences will appear here"
	gh["data"] = "Details will appear here"
	if err := c.Request.ParseForm(); err != nil {
		gh["buttons"] = "Error loading the form.\n" + err.Error()
		c.HTML(500, "differences.html", gh)
		return
	}
	diffs, err := a.diffMan.AllDiffs()
	if err != nil {
		gh["buttons"] = "Could not load this.\n" + err.Error()
		gh["data"] = "Error loading this.\n" + err.Error()
		c.HTML(500, "differences.html", gh)
		return
	}
	form := map[string][]string(c.Request.Form)
	{
		buttons := make([]s.HtmlInter, len(*diffs))
		for i, d := range *diffs {
			buttons[i] = s.NewHtmlButton2(d.Id, d.SimpleString())
		}
		if len(buttons) > 0 {
			tmp := s.NewHtmlCollection()
			for _, b := range buttons {
				tmp.Add(b)
				tmp.Add(s.NewHtmlBr())
			}
			gh["buttons"] = tmp.Template()
		}
	}
	if len(form) > 0 {
		var res string
		for diffId, _ := range form {
			if diffId == "button_delete" {
				a.diffMan.Delete(a.diffMan.CurrentDisplay)
				a.diffMan.CurrentDisplay = ""
				c.Redirect(307, "/diff")
				return
			} else {
				for _, diff := range *diffs {
					if diff.Id == diffId {
						res = a.diffMan.GenerateReport(&diff) + "\n"
						a.diffMan.CurrentDisplay = diffId
						break
					}
				}
			}
		}
		gh["data"] = s.NewHtmlCollection(s.NewHtmlBasic("pre", res), s.NewHtmlBasic("form", s.NewHtmlButton("Delete").String())).Template()
	}
	c.HTML(200, "differences.html", gh)
}
