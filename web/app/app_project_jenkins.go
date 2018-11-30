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
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/venicegeo/vzutil-versioning/web/app/structs/html"
	u "github.com/venicegeo/vzutil-versioning/web/util"
)

func (a *Application) jenkinsTesting(c *gin.Context) {
	var form struct {
		Back       string `form:"button_back"`
		JenkinsUrl string `form:"url"`
		RepoId     string `form:"repo"`
		Submit     string `form:"button_submit"`
	}
	if err := c.Bind(&form); err != nil {
		c.String(400, "Unable to bind form: %s", err.Error())
		return
	}
	projId := c.Param("proj")
	if form.Back != "" {
		c.Redirect(303, "/project/"+projId)
		return
	}

	if form.Submit != "" {
		if form.RepoId == "" {
			c.String(400, "You must select a repository")
			return
		}
		_, err := a.jnknsMngr.Add(projId, form.RepoId, form.JenkinsUrl)
		if err != nil {
			c.String(400, "Error adding this url: %s", err.Error())
			return
		}
	}
	project, err := a.rtrvr.GetProjectById(projId)
	if err != nil {
		c.String(500, "Unable to get the project: %s", err)
		return
	}
	repos, err := project.GetAllRepositories()
	if err != nil {
		c.String(500, "Unable to get the repos: %s", err)
		return
	}
	availableRepos := map[string]string{}
	idsToNames := map[string]string{}
	for _, r := range repos {
		availableRepos[r.Id] = r.Fullname
		idsToNames[r.Id] = r.Fullname
	}
	h := gin.H{}

	entries, err := a.jnknsMngr.getAllEntriesProj(projId)
	inuseBuf := bytes.NewBufferString("")
	tempBuf := bytes.NewBufferString("")
	if err != nil {
		inuseBuf.WriteString("Error: ")
		inuseBuf.WriteString(err.Error())
	} else {
		for _, entry := range entries {
			delete(availableRepos, entry.RepositoryId)
			inuseBuf.WriteString(idsToNames[entry.RepositoryId])
			inuseBuf.WriteString("\n")

			data, err := a.jnknsMngr.GetOrgsAndSpaces(entry.Id)
			fmt.Println(data)
			tempBuf.WriteString(u.Format("%#v %s\n", data, err))
		}
	}
	drop := structs.NewHtmlDropdown("repo")
	for id, name := range availableRepos {
		drop.Add(id, name)
	}
	h["repo_dropdown"] = drop.Template()
	h["inuse"] = inuseBuf.String()
	h["temp"] = tempBuf.String()
	c.HTML(200, "jenkins.html", h)
}
