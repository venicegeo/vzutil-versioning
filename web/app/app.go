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
	"errors"
	"html/template"
	"log"
	"os"
	"os/exec"
	"regexp"

	"github.com/gin-gonic/gin"
	"github.com/venicegeo/pz-gocommon/elasticsearch"
	h "github.com/venicegeo/vzutil-versioning/web/app/helpers"
	s "github.com/venicegeo/vzutil-versioning/web/app/structs"
	u "github.com/venicegeo/vzutil-versioning/web/util"
)

type Application struct {
	indexName      string
	singleLocation string
	debugMode      bool

	wrkr     *h.Worker
	rprtr    *h.Reporter
	diffMan  *h.DifferenceManager
	killChan chan bool
}

type Back struct {
	BackButton string `form:"button_back"`
}

func NewApplication(indexName, singleLocation string, debugMode bool) *Application {
	return &Application{
		indexName:      indexName,
		singleLocation: singleLocation,
		debugMode:      debugMode,
		killChan:       make(chan bool),
	}
}

func (a *Application) Start() chan error {
	done := make(chan error)
	log.Println("Starting up...")

	if err := a.handleMaven(); err != nil {
		log.Fatal(err)
	}

	url, user, pass, err := s.GetVcapES()
	log.Printf("The elasticsearch url has been found to be [%s]\n", url)
	if err != nil {
		log.Fatal(err)
	}
	i, err := elasticsearch.NewIndex2(url, user, pass, a.indexName, `
{
	"mappings": {
		"project":{
			"dynamic":"strict",
			"properties":{
				"full_name":{"type":"text"},
				"name":{"type":"text"},
				"last_sha":{"type":"text"},
				"webhook_order":{"type":"text"},
				"tag_shas":{"type":"text"},
				"entries":{"type":"text"}
			}
		},
		"dependency":{
			"dynamic":"strict",
			"properties":{
				"hashsum":{"type":"text"},
				"name":{"type":"text"},
				"version":{"type":"text"},
				"language":{"type":"text"}
			}
		},
		"difference":{
			"dynamic":"strict",
			"properties":{
				"full_name":{"type":"text"},
				"old_sha":{"type":"text"},
				"new_sha":{"type":"text"},
				"removed":{"type":"text"},
				"added":{"type":"text"},
				"time":{"type":"long"}
			}
		}
	}
}`)
	if err != nil {
		log.Fatal(err.Error())
	} else {
		log.Println(i.GetVersion())
	}

	a.diffMan = h.NewDifferenceManager(i)
	a.wrkr = h.NewWorker(i, a.singleLocation, 10, a.diffMan)
	a.wrkr.Start()
	a.rprtr = h.NewReporter(i)

	port := os.Getenv("PORT")
	if port == "" {
		port = "20012"
	}

	log.Println("Starting on port", port)
	server := u.Server{}
	server.Configure([]u.RouteData{
		u.RouteData{"GET", "/", a.defaultPath},
		u.RouteData{"POST", "/webhook", a.webhookPath},
		u.RouteData{"GET", "/generate/tags/:org/:repo", a.updateAllTags},
		u.RouteData{"GET", "/generate/tags/:org", a.updateAllTagsOrg},
		u.RouteData{"GET", "/generate/sha/:org/:repo/:sha", a.specificSha},

		u.RouteData{"GET", "/report/sha/:org/:repo/:sha", a.reportSha},
		u.RouteData{"GET", "/report/tag/:tagorg", a.reportTag},
		u.RouteData{"GET", "/report/tag/:tagorg/:tagrepo", a.reportTag},
		u.RouteData{"GET", "/report/tag/:tagorg/:tagrepo/:tag", a.reportTag},

		u.RouteData{"GET", "/list/shas/:org/:repo", a.listShas},
		u.RouteData{"GET", "/list/tags/:org/:repo", a.listTagsRepo},
		u.RouteData{"GET", "/list/tags/:org", a.listTags},
		u.RouteData{"GET", "/list/projects", a.listProjects},
		u.RouteData{"GET", "/list/projects/:org", a.listProjectsOrg},

		u.RouteData{"GET", "/ui", a.formPath},
		u.RouteData{"GET", "/diff", a.diffPath},
	})
	select {
	case err = <-server.Start(":" + port):
		done <- err
	case <-a.killChan:
		done <- errors.New(u.Format("was stopped: %s", server.Stop()))
	}
	return done
}
func (a *Application) Stop() {
	a.killChan <- true
}

func (a *Application) defaultPath(c *gin.Context) {
	c.String(200, "Welcome to the dependency service!")
}

func (a *Application) formPath(c *gin.Context) {
	var form s.Form
	if err := c.Bind(&form); err != nil {
		c.String(400, err.Error())
		return
	}
	if form.IsEmpty() {
		ps, err := a.rprtr.ListProjects()
		h := gin.H{}
		if err != nil {
			h["projects"] = "Sorry... could not\nload this.\n" + err.Error()
		} else {
			res := ""
			for i, p := range ps {
				if i > 0 {
					res += "\n"
				}
				res += p
			}
			h["projects"] = res
		}
		diffs, err := a.diffMan.DiffList(250)
		if err != nil {
			h["differences"] = "Sorry... could not\nload this.\n" + err.Error()
		} else {
			res := ""
			for i, d := range diffs {
				if i > 0 {
					res += "\n"
				}
				res += d
			}
			h["differences"] = res
		}
		c.HTML(200, "form.html", h)
		return
	}
	buttonPress := form.FindButtonPress()
	switch buttonPress {
	case s.ReportTag:
		if form.ReportTagRepo != "" && form.ReportTagOrg == "" {
			a.displayFailure(c, "Must specify an org if you specify a repo")
		} else {
			url := "/report/tag"
			if form.ReportTagOrg != "" {
				url += "/" + form.ReportTagOrg
			}
			if form.ReportTagRepo != "" {
				url += "/" + form.ReportTagRepo
			}
			if form.ReportTagTag != "" {
				url += "/" + form.ReportTagTag
			}
			c.Redirect(307, url)
		}
	case s.ReportSha:
		c.Redirect(307, u.Format("/report/sha/%s/%s/%s", form.ReportShaOrg, form.ReportShaRepo, form.ReportShaSha))
	case s.ListTags:
		if form.TagsRepo != "" {
			c.Redirect(307, u.Format("/list/tags/%s/%s", form.TagsOrg, form.TagsRepo))
		} else {
			c.Redirect(307, "/list/tags/"+form.TagsOrg)
		}
	case s.ListShas:
		c.Redirect(307, u.Format("/list/shas/%s/%s", form.ShasOrg, form.ShasRepo))
	case s.GenerateTag:
		if form.AllTagRepo != "" {
			c.Redirect(307, u.Format("/generate/tags/%s/%s", form.AllTagOrg, form.AllTagRepo))
		} else {
			c.Redirect(307, "/generate/tags/"+form.AllTagOrg)
		}
	case s.GenerateSha:
		c.Redirect(307, u.Format("/generate/sha/%s/%s/%s", form.ByShaOrg, form.ByShaRepo, form.ByShaSha))
	case s.Differences:
		c.Redirect(307, "/diff")
	default:
		c.String(400, "What did you do? :(")
	}
}
func (a *Application) diffPath(c *gin.Context) {
	if a.checkBack(c) {
		return
	}
	gh := gin.H{}
	if err := c.Request.ParseForm(); err != nil {
		gh["buttons"] = "Error loading the form.\n" + err.Error()
		c.HTML(500, "differences.html", gh)
		return
	}
	form := map[string][]string(c.Request.Form)
	{
		diffs, err := a.diffMan.DiffList(250)
		if err != nil {
			gh["buttons"] = "Could not load this.\n" + err.Error()
			c.HTML(500, "differences.html", gh)
			return
		}
		buttons := make([]template.HTML, len(diffs))
		for i, d := range diffs {
			buttons[i] = s.NewHtmlButton(d).Template()
		}
		gh["buttons"] = buttons
	}
	{
		var diffs *[]h.Difference
		var err error
		if len(form) > 0 {
			if diffs, err = a.diffMan.AllDiffs(250); err != nil {
				gh["data"] = "Error loading this.\n" + err.Error()
				c.HTML(500, "differences.html", gh)
				return
			}
		}
		res := ""
		for _, value := range form {
			if len(value) == 0 {
				continue
			}
			for _, diff := range *diffs {
				if diff.SimpleString() == value[0] {
					res += a.diffMan.GenerateReport(&diff) + "\n"
				}
			}
		}
		gh["data"] = res
	}
	c.HTML(200, "differences.html", gh)
}

func (a *Application) checkBack(c *gin.Context) (wasHandled bool) {
	var back Back
	if err := c.Bind(&back); err != nil {
		c.String(500, err.Error())
		return true
	}
	if back.BackButton != "" {
		c.Redirect(307, "/ui")
		return true
	}
	return false
}

func (a *Application) displaySuccess(c *gin.Context, data string) {
	if !a.checkForRedirect(c) {
		c.String(200, data)
	} else {
		c.HTML(200, "back.html", gin.H{"data": data})
	}
}
func (a *Application) displayFailure(c *gin.Context, data string) {
	//TODO assuming 400
	if !a.checkForRedirect(c) {
		c.String(400, data)
	} else {
		c.HTML(400, "back.html", gin.H{"data": data})
	}
}

func (a *Application) checkForRedirect(c *gin.Context) bool {
	return c.Request.Header.Get("Referer") != ""
}

func (a *Application) handleMaven() error {
	_, err := os.Stat("settings.xml")
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		} else {
			return err
		}
	}
	dat, _ := exec.Command("mvn", "-X").Output()
	re := regexp.MustCompile(`Reading user settings from (.+)\/`)
	finds := re.FindStringSubmatch(string(dat))
	if len(finds) != 2 {
		return errors.New("Couldnt find maven settings location")
	}

	return exec.Command("mv", "settings.xml", finds[1]).Run()
}
