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
	"github.com/gin-gonic/gin"
)

func (a *Application) jenkinsTesting(c *gin.Context) {
	var form struct {
		Back         string `form:"button_back"`
		JenkinsUrl   string `form:"url"`
		RepoFullname string `form:"repo"`
		Submit       string `form:"button_submit"`
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
	if form.Submit == "" {
		c.HTML(200, "jenkins.html", nil)
		return
	}
	_, err := a.jnknsMngr.Add(projId, form.RepoFullname, form.JenkinsUrl)
	if err != nil {
		c.String(400, "Error adding this url: %s", err.Error())
		return
	}
	c.Redirect(303, "/project/"+projId)
}
