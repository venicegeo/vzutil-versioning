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
	"html/template"

	"github.com/gin-gonic/gin"
	s "github.com/venicegeo/vzutil-versioning/web/app/structs"
)

func (a *Application) textDiffPath(c *gin.Context) {
	type TDiff struct {
		Ui       string `form:"button_back"`
		Actual   string `form:"actual"`
		Expected string `form:"expected"`
		Compare  string `form:"button_textdiff"`

		Load     string   `form:"button_loadrepos"`
		Repos    []string `form:"repos[]"`
		Checkout []string `form:"checkout[]"`
		Plural   string   `form:"button_plural"`
	}
	var tmp TDiff
	if err := c.Bind(&tmp); err != nil {
		c.String(400, "Error binding form: %s", err.Error())
		return
	}
	h := gin.H{
		"result":    "Compare Results will appear here",
		"actual":    tmp.Actual,
		"expected":  tmp.Expected,
		"loadRepos": []string{"", ""},
	}
	if tmp.Load != "" {
		repos, err := a.rtrvr.ListRepositories()
		if err != nil {
			c.String(400, "Error collecting repository list: %s", err.Error())
			return
		}
		load := make([]string, 0, len(repos)*2)
		for _, proj := range repos {
			load = append(load, proj)
			load = append(load, "")
		}
		h["loadRepos"] = load
		c.HTML(200, "textdiff.html", h)
	} else if tmp.Ui != "" {
		c.Redirect(303, "/ui")
	} else if tmp.Plural != "" {
		repos, err := a.plrlRnnr.RunAgainstPluralStr(tmp.Repos, tmp.Checkout)
		if err != nil {
			c.String(400, "Error running against pural: %s", err.Error())
			return
		}
		h["actual"] = repos
		c.HTML(200, "textdiff.html", h)
	} else if tmp.Compare != "" {
		res, err := a.cmprRnnr.CompareStrings(tmp.Actual, tmp.Expected)
		if err != nil {
			h["result"] = "Error running this: " + err.Error()
			c.HTML(400, "textdiff.html", h)
			return
		}
		h["result"] = res
		c.HTML(200, "textdiff.html", h)
	} else {
		c.HTML(200, "textdiff.html", h)
	}
}

func (a *Application) customDiffPath(c *gin.Context) {
	type CDiff struct {
		Ui      string `form:"button_back"`
		Org     string `form:"cdifforg"`
		Repo    string `form:"cdiffrepo"`
		ShaOld  string `form:"cdiffshaold"`
		ShaNew  string `form:"cdiffshanew"`
		Compare string `form:"button_cdiff"`
	}
	var tmp CDiff
	if err := c.Bind(&tmp); err != nil {
		c.String(400, "Error binding form: %s", err.Error())
		return
	}
	h := gin.H{
		"data":        "Compare Results will appear here",
		"cdifforg":    tmp.Org,
		"cdiffrepo":   tmp.Repo,
		"cdiffshaold": tmp.ShaOld,
		"cdiffshanew": tmp.ShaNew,
	}
	if tmp.Ui != "" {
		c.Redirect(303, "/ui")
	} else if tmp.Compare != "" {
		diff, err := a.diffMan.ShaCompare(tmp.Org+"/"+tmp.Repo, tmp.ShaOld, tmp.ShaNew)
		if err != nil {
			h["data"] = err.Error()
			c.HTML(500, "customdiff.html", h)
			return
		}
		if diff == nil {
			h["data"] = "There are no differences."
		} else {
			h["data"] = a.diffMan.GenerateReport(diff)
		}
		c.HTML(200, "customdiff.html", h)
	} else {
		c.HTML(200, "customdiff.html", h)
	}
}

func (a *Application) diffPath(c *gin.Context) {
	if a.checkBack(c) {
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
				tmp.Add(&s.HtmlBr{})
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
		gh["data"] = template.HTML(s.NewHtmlCollection(s.NewHtmlBasic("pre", res), s.NewHtmlBasic("form", s.NewHtmlButton("Delete").String())).Template())
	}
	c.HTML(200, "differences.html", gh)
}
