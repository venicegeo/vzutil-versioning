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
	searchSize     int
	singleLocation string
	debugMode      bool

	wrkr     *h.Worker
	rprtr    *h.Reporter
	diffMan  *h.DifferenceManager
	killChan chan bool

	index *elasticsearch.Index
}

type Back struct {
	BackButton string `form:"button_back"`
}

func NewApplication(indexName, singleLocation string, debugMode bool) *Application {
	return &Application{
		indexName:      indexName,
		searchSize:     250,
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
				"full_name":{"type":"keyword"},
				"name":{"type":"keyword"},
				"tag_shas":{
					"dynamic":"strict",
					"properties":{
						"tag":{"type":"keyword"},
						"sha":{"type":"keyword"}
					}
				},
				"refs":{
					"dynamic":"strict",
					"properties":{
						"name":{"type":"keyword"},
						"webhook_order":{"type":"keyword"},
						"entries":{
							"dynamic":"strict",
							"properties":{
								"sha":{"type":"keyword"},
								"entry_reference":{"type":"keyword"},
								"dependencies":{"type":"keyword"}
							}
						}
					}
				}
			}
		},
		"dependency":{
			"dynamic":"strict",
			"properties":{
				"hashsum":{"type":"keyword"},
				"name":{"type":"keyword"},
				"version":{"type":"keyword"},
				"language":{"type":"keyword"}
			}
		},
		"difference":{
			"dynamic":"strict",
			"properties":{
				"id":{"type":"keyword"},
				"full_name":{"type":"keyword"},
				"old_sha":{"type":"keyword"},
				"new_sha":{"type":"keyword"},
				"removed":{"type":"keyword"},
				"added":{"type":"keyword"},
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
	a.index = i

	a.diffMan = h.NewDifferenceManager(i)
	a.wrkr = h.NewWorker(i, a.singleLocation, 4, a.diffMan)
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

		u.RouteData{"GET", "/report/sha/:org/:repo/:sha", a.reportSha},
		u.RouteData{"GET", "/report/tag/:tagorg", a.reportTag},
		u.RouteData{"GET", "/report/tag/:tagorg/:tagrepo", a.reportTag},
		u.RouteData{"GET", "/report/tag/:tagorg/:tagrepo/:tag", a.reportTag},

		u.RouteData{"GET", "/list/shas/:org/:repo", a.listShas},
		u.RouteData{"GET", "/list/refs/:org/:repo", a.listRefsRepo},
		u.RouteData{"GET", "/list/refs/:org", a.listRefs},
		u.RouteData{"GET", "/list/projects", a.listProjects},
		u.RouteData{"GET", "/list/projects/:org", a.listProjectsOrg},

		u.RouteData{"GET", "/search", a.uiSearchForDep},
		u.RouteData{"GET", "/search/:dep", a.searchForDep},
		u.RouteData{"GET", "/search/:dep/:version", a.searchForDep},

		u.RouteData{"GET", "/ui", a.formPath},

		u.RouteData{"GET", "/diff", a.diffPath},
		u.RouteData{"GET", "/cdiff", a.customDiffPath},
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
		diffs, err := a.diffMan.DiffList(a.searchSize)
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
	case s.DepSearch:
		c.Redirect(307, "/search")
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
	case s.ListRefs:
		if form.RefsRepo != "" {
			c.Redirect(307, u.Format("/list/refs/%s/%s", form.RefsOrg, form.RefsRepo))
		} else {
			c.Redirect(307, "/list/refs/"+form.RefsOrg)
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
	case s.CustomDifference:
		c.Redirect(307, "/cdiff")
	default:
		c.String(400, "What did you do? :(")
	}
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
