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

package util

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"

	"github.com/gin-gonic/gin"
	"github.com/venicegeo/pz-gocommon/elasticsearch"
	nt "github.com/venicegeo/pz-gocommon/gocommon"
)

type Application struct {
	indexName      string
	singleLocation string
	debugMode      bool

	wrkr     *Worker
	rprtr    *Reporter
	killChan chan bool
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
	fmt.Println("Starting up...")

	if err := a.handleMaven(); err != nil {
		log.Fatal(err)
	}

	url, user, pass, err := GetVcapES()
	fmt.Printf("The elasticsearch url has been found to be [%s]\n", url)
	if user != "" {
		fmt.Println("There is a username")
	}
	if pass != "" {
		fmt.Println("There is a password")
	}
	if err != nil {
		log.Fatal(err)
	}
	i, err := elasticsearch.NewIndex2(url, user, pass, a.indexName, `
{
	"mappings": {
		"project":{
			"dynamic":"strict",
			"properties":{
				"full_name":{"type":"string"},
				"name":{"type":"string"},
				"last_sha":{"type":"string"},
				"tag_shas":{"type":"string"},
				"entries":{"type":"string"}
			}
		},
		"dependency":{
			"dynamic":"strict",
			"properties":{
				"hashsum":{"type":"string"},
				"name":{"type":"string"},
				"version":{"type":"string"},
				"language":{"type":"string"}
			}
		}
	}
}`)
	if err != nil {
		log.Fatal(err.Error())
	} else {
		fmt.Println(i.GetVersion())
	}

	a.wrkr = NewWorker(i, a.singleLocation)
	a.wrkr.Start()
	a.rprtr = NewReporter(i)

	port := os.Getenv("PORT")
	if port == "" {
		port = "20012"
	}

	fmt.Println("Starting on port", port)
	server := Server{}
	server.Configure([]RouteData{
		RouteData{"GET", "/", a.defaultPath},
		RouteData{"POST", "/webhook", a.webhookPath},
		RouteData{"GET", "/generate/tags/:org/:repo", a.updateAllTags},
		RouteData{"GET", "/generate/sha/:org/:repo/:sha", a.specificSha},
		RouteData{"GET", "/report/sha/:org/:repo/:sha", a.reportSha},
		RouteData{"GET", "/report/tag/:tag", a.reportTag},
	})
	select {
	case err = <-server.Start(":" + port):
		done <- err
	case <-a.killChan:
		done <- fmt.Errorf("was stopped: %s", server.Stop())
	}
	return done
}
func (a *Application) Stop() {
	a.killChan <- true
}

func (a *Application) defaultPath(c *gin.Context) {
	c.String(200, "Welcome to the dependency service!")
}
func (a *Application) webhookPath(c *gin.Context) {
	git := GitWebhook{}

	if err := c.BindJSON(&git); err != nil {
		log.Println("Unable to bind json:", err.Error())
		c.Status(400)
		return
	}

	fmt.Println(git.Repository.FullName, git.AfterSha, git.Ref)
	c.String(200, "Thanks!")

	a.wrkr.AddTask(&git)
}

func (a *Application) updateAllTags(c *gin.Context) {
	name := c.Param("repo")
	fullName := fmt.Sprintf("%s/%s", c.Param("org"), name)
	tr := newTagsRunner(name, fullName)
	dat, err := tr.run()
	if err != nil {
		c.String(400, "Sorry, no can do. Problem: [%s]", err.Error())
		return
	}
	go func() {
		for sha, ref := range dat {
			git := GitWebhook{
				Ref:      ref,
				AfterSha: sha,
				Repository: GitRepository{
					Name:     name,
					FullName: fullName,
				},
			}
			fmt.Println(git.Repository.FullName, git.AfterSha, git.Ref)
			a.wrkr.AddTask(&git)
		}
	}()
	c.String(200, "Yeah, I can do that. Check back in a minute")
}

func (a *Application) specificSha(c *gin.Context) {
	name := c.Param("repo")
	fullName := fmt.Sprintf("%s/%s", c.Param("org"), name)
	sha := c.Param("sha")
	code, _, _, err := nt.HTTP(nt.HEAD, fmt.Sprintf("https://github.com/%s/commit/%s", fullName, sha), nt.NewHeaderBuilder().GetHeader(), nil)
	if err != nil {
		c.String(500, "could not verify this sha:", err.Error())
		return
	}
	if code != 200 {
		c.String(400, "could not verify this sha, head code:", code)
		return
	}

	c.String(200, "I got this, check back in a bit")

	git := GitWebhook{
		AfterSha: sha,
		Repository: GitRepository{
			Name:     name,
			FullName: fullName,
		},
	}
	fmt.Println(git.Repository.FullName, git.AfterSha)
	a.wrkr.AddTask(&git)
}

func (a *Application) reportSha(c *gin.Context) {
	fullName := fmt.Sprintf("%s/%s", c.Param("org"), c.Param("repo"))
	sha := c.Param("sha")
	deps, err := a.rprtr.ReportBySha(fullName, sha)
	if err != nil {
		c.String(400, "Unable to do this:", err.Error())
	}
	res := "Report for " + fullName + " at " + sha + "\n===================="
	for _, dep := range deps {
		res += "\n" + dep
	}
	c.String(200, res)
}
func (a *Application) reportTag(c *gin.Context) {
	tag := c.Param("tag")
	deps, err := a.rprtr.ReportByTag(tag)
	if err != nil {
		c.String(500, "Unable to do this: ", err.Error())
		return
	}
	res := ""
	for name, depss := range deps {
		res += "====================\n"
		res += name + " at " + tag + "\n"
		res += "===================="
		for _, dep := range depss {
			res += "\n" + dep
		}
	}
	c.String(200, res)
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
		return fmt.Errorf("Couldnt find maven settings location")
	}

	return exec.Command("mv", "settings.xml", finds[1]).Run()
}
