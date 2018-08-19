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
	"github.com/venicegeo/vzutil-versioning/web/es"
	u "github.com/venicegeo/vzutil-versioning/web/util"
)

type Application struct {
	singleLocation  string
	compareLocation string
	debugMode       bool

	server *u.Server

	wrkr     *Worker
	rtrvr    *Retriever
	diffMan  *DifferenceManager
	ff       *FireAndForget
	cmprRnnr *CompareRunner

	killChan chan bool

	index elasticsearch.IIndex
}

const ESMapping = `
{
	"mappings": {
		"` + RepositoryEntryType + `": ` + RepositoryDependencyScanMapping + `,
		"` + DifferenceType + `": ` + DifferenceMapping + `,
		"` + ProjectEntryType + `": ` + es.ProjectEntryMapping + `,
		"` + ProjectType + `": ` + es.ProjectMapping + `
	}
}`
const RepositoryEntryType = `repository_entry`
const DifferenceType = `difference`
const ProjectEntryType = `project_entry`
const ProjectType = `project`

type Back struct {
	BackButton string `form:"button_back"`
}

func NewApplication(index elasticsearch.IIndex, singleLocation, compareLocation string, debugMode bool) *Application {
	return &Application{
		index:           index,
		singleLocation:  singleLocation,
		compareLocation: compareLocation,
		debugMode:       debugMode,
		killChan:        make(chan bool),
	}
}

func (a *Application) StartInternals() {
	log.Println("Starting internals...")

	if err := a.handleMaven(); err != nil {
		log.Fatalln(err)
	}

	a.diffMan = NewDifferenceManager(a)
	a.wrkr = NewWorker(a, 2)
	a.rtrvr = NewRetriever(a)
	a.ff = NewFireAndForget(a)
	a.cmprRnnr = NewCompareRunner(a)

	a.wrkr.Start()

	a.server = u.NewServer()
	if _, err := os.Stat("localhost.crt"); err == nil {
		if _, err = os.Stat("localhost.key"); err == nil {
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
		u.RouteData{"GET", "/addrepo/:proj", a.addRepoToProject, true},
		u.RouteData{"POST", "/addrepo/:proj", a.addRepoToProject, true},
		u.RouteData{"GET", "/genbranch/:proj/:org/:repo", a.generateBranch, true},
		u.RouteData{"GET", "/reportref/:proj", a.reportRefOnProject, true},
		u.RouteData{"GET", "/removerepo/:proj", a.removeReposFromProject, true},
		u.RouteData{"GET", "/depsearch/:proj", a.searchForDepInProject, true},
		u.RouteData{"GET", "/depsearch", a.searchForDep, true},
		u.RouteData{"GET", "/diff/:proj", a.differencesInProject, true},
		u.RouteData{"GET", "/reportsha", a.reportSha, true},
		u.RouteData{"GET", "/cdiff", a.customDiff, true},
		u.RouteData{"POST", "/cdiff", a.customDiff, true},
	})
}

func (a *Application) StartServer() chan error {
	done := make(chan error)

	port := os.Getenv("PORT")
	if port == "" {
		port = "20012"
	}

	log.Printf("Starting server on port %s...\n", port)

	select {
	case err := <-a.server.Start(":" + port):
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
