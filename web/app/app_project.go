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
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	nt "github.com/venicegeo/pz-gocommon/gocommon"
	s "github.com/venicegeo/vzutil-versioning/web/app/structs"
	"github.com/venicegeo/vzutil-versioning/web/es"
	u "github.com/venicegeo/vzutil-versioning/web/util"
)

func (a *Application) viewProject(c *gin.Context) {
	proj := c.Param("proj")
	var form struct {
		Back   string `form:"button_back"`
		Util   string `form:"button_util"`
		Sha    string `form:"button_sha"`
		Gen    string `form:"button_gen"`
		Diff   string `form:"button_diff"`
		Reload string `form:"button_reload"`
	}
	if err := c.Bind(&form); err != nil {
		c.String(400, "Unable to bind form: %s", err.Error())
		return
	}
	depsStr := "Result info will appear here"
	if form.Back != "" {
		c.Redirect(303, "/ui")
		return
	} else if form.Reload != "" {
		c.Redirect(303, "/project/"+proj)
		return
	} else if form.Util != "" {
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
		scan, found, err := a.rtrvr.ScanBySha(form.Sha)
		if !found && err != nil {
			c.String(400, "Unable to find this sha: %s", err.Error())
			return
		} else if found && err != nil {
			c.String(500, "Unable to obtain the results: %s", err.Error())
			return
		}
		depsStr = a.reportAtShaWrk(scan)
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
	mux := sync.Mutex{}
	errs := make(chan error, len(repos))
	work := func(repoName string) {
		refs, err := a.rtrvr.ListRefsRepo(repoName)
		if err != nil {
			errs <- err
			return
		}
		tempAccord := s.NewHtmlAccordion()
		shas, _, err := a.rtrvr.ListShas(repoName)
		if err != nil {
			errs <- err
			return
		}
		for _, ref := range refs {
			c := s.NewHtmlCollection()
			correctShas := shas["refs/"+ref]
			for i, sha := range correctShas {
				c.Add(s.NewHtmlButton2("button_sha", sha))
				if i < len(correctShas)-1 {
					c.Add(s.NewHtmlBr())
				}
			}
			tempAccord.AddItem(ref, s.NewHtmlForm(c).Post())
		}
		mux.Lock()
		accord.AddItem(repoName, s.NewHtmlCollection(s.NewHtmlForm(s.NewHtmlButton2("button_gen", "Generate Branch - "+repoName)).Post(), tempAccord.Sort()))
		mux.Unlock()
		errs <- nil
	}
	for _, repoName := range repos {
		go work(repoName)
	}
	err = nil
	for i := 0; i < len(repos); i++ {
		e := <-errs
		if e != nil {
			err = e
		}
	}
	if err != nil {
		c.String(500, "Error retrieving data: %s", err.Error())
		return
	}
	accord.Sort()
	h := gin.H{}
	h["accordion"] = accord.Template()
	h["deps"] = depsStr
	{
		diffs, err := a.diffMan.DiffListInProject(proj)
		if err != nil {
			h["diff"] = ""
		} else {
			h["diff"] = u.Format(" (%d)", len(diffs))
		}
	}
	c.HTML(200, "project.html", h)
}

func (a *Application) addReposToProject(c *gin.Context) {
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

func (a *Application) removeReposFromProject(c *gin.Context) {
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
		if resp.Hits.TotalHits != 1 {
			c.String(400, "Could not find the project entry")
			return
		}
		_, err = a.index.DeleteByIDWait("project_entry", resp.Hits.Hits[0].Id)
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
