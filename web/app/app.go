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
	s "github.com/venicegeo/vzutil-versioning/web/app/structs"
	u "github.com/venicegeo/vzutil-versioning/web/util"
)

type Application struct {
	indexName       string
	singleLocation  string
	compareLocation string
	debugMode       bool

	server *u.Server

	wrkr     *Worker
	rtrvr    *Retriever
	diffMan  *DifferenceManager
	wbhkRnnr *WebhookRunner
	cmprRnnr *CompareRunner

	killChan chan bool

	index *elasticsearch.Index
}

type Back struct {
	BackButton string `form:"button_back"`
}

func NewApplication(indexName, singleLocation, compareLocation string, debugMode bool) *Application {
	return &Application{
		indexName:       indexName,
		singleLocation:  singleLocation,
		compareLocation: compareLocation,
		debugMode:       debugMode,
		killChan:        make(chan bool),
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
		"repository_entry":{
			"dynamic":"strict",
			"properties":{
				"repo_fullname":{"type":"keyword"},
				"repo_name":{"type":"keyword"},
				"ref_name":{"type":"keyword"},
				"sha":{"type":"keyword"},
				"timestamp":{"type":"long"},
				"dependencies":{"type":"keyword"}
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
				"ref":{"type":"keyword"},
				"old_sha":{"type":"keyword"},
				"new_sha":{"type":"keyword"},
				"removed":{"type":"keyword"},
				"added":{"type":"keyword"},
				"time":{"type":"long"}
			}
		},
		"project_entry": {
			"dynamic":"strict",
			"properties":{
				"name":{"type":"keyword"},
				"repo":{"type":"keyword"}
			}
		},
		"project":{
			"dynamic":"strict",
			"properties":{
				"name":{"type":"keyword"},
				"displayname":{"type":"keyword"}
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

	a.diffMan = NewDifferenceManager(a)
	a.wrkr = NewWorker(a, 2)
	a.rtrvr = NewRetriever(a)
	a.wbhkRnnr = NewWebhookRunner(a)
	a.cmprRnnr = NewCompareRunner(a)

	a.wrkr.Start()

	port := os.Getenv("PORT")
	if port == "" {
		port = "20012"
	}

	log.Println("Starting on port", port)
	a.server = u.NewServer()
	if _, err := os.Stat("crt"); err != nil {
		if _, err = os.Stat("key"); err != nil {
			a.server.SetTLSInfo("localhost.crt", "localhost.key")
		}
	}
	a.server.Configure([]u.RouteData{
		u.RouteData{"GET", "/", a.defaultPath, false},
		u.RouteData{"POST", "/webhook", a.webhookPath, false},

		u.RouteData{"GET", "/login", a.login, false},
		u.RouteData{"POST", "/login", a.login, false},

		u.RouteData{"GET", "/ui", a.projectsOverview, true},
		u.RouteData{"GET", "/newproj", a.newProject, true},
		u.RouteData{"POST", "/newproj", a.newProject, true},
		u.RouteData{"GET", "/project/:proj", a.viewProject, true},
		u.RouteData{"POST", "/project/:proj", a.viewProject, true},
		u.RouteData{"GET", "/addrepo/:proj", a.addReposToProject, true},
		u.RouteData{"POST", "/addrepo/:proj", a.addReposToProject, true},
		u.RouteData{"GET", "/genbranch/:proj/:org/:repo", a.generateBranch, true},
		u.RouteData{"GET", "/reportref/:proj", a.reportRefOnProject, true},
		u.RouteData{"GET", "/removerepo/:proj", a.removeReposFromProject, true},
		u.RouteData{"GET", "/depsearch/:proj", a.searchForDepInProject, true},
		u.RouteData{"GET", "/depsearch", a.searchForDep, true},
		u.RouteData{"GET", "/diff/:proj", a.differencesInProject, true},
		u.RouteData{"GET", "/reportsha", a.reportSha, true},
	})
	select {
	case err = <-a.server.Start(":" + port):
		done <- err
	case <-a.killChan:
		done <- errors.New(u.Format("was stopped: %s", a.server.Stop()))
	}
	return done
}
func (a *Application) Stop() {
	a.killChan <- true
}

func (a *Application) defaultPath(c *gin.Context) {
	c.String(200, "Welcome to the dependency service!")
}

func (a *Application) login(c *gin.Context) {
	var form struct {
		Key    string `form:"key"`
		Submit string `form:"button_submit"`
	}
	if err := c.Bind(&form); err != nil {
		c.String(400, "Unable to bind form")
		return
	}
	if form.Submit == "" {
		c.HTML(200, "login.html", nil)
	} else {
		a.server.CreateAuth(c)
		c.Redirect(302, "/ui")
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
