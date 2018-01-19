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
	"errors"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/venicegeo/pz-gocommon/elasticsearch"
	nt "github.com/venicegeo/pz-gocommon/gocommon"
	"github.com/venicegeo/vzutil-versioning/web/f"
	"github.com/venicegeo/vzutil-versioning/web/util/table"
)

type Application struct {
	indexName      string
	singleLocation string
	debugMode      bool

	wrkr     *Worker
	rprtr    *Reporter
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

	url, user, pass, err := GetVcapES()
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
		log.Println(i.GetVersion())
	}

	a.wrkr = NewWorker(i, a.singleLocation, 10)
	a.wrkr.Start()
	a.rprtr = NewReporter(i)

	port := os.Getenv("PORT")
	if port == "" {
		port = "20012"
	}

	log.Println("Starting on port", port)
	server := Server{}
	server.Configure([]RouteData{
		RouteData{"GET", "/", a.defaultPath},
		RouteData{"POST", "/webhook", a.webhookPath},
		RouteData{"GET", "/generate/tags/:org/:repo", a.updateAllTags},
		RouteData{"GET", "/generate/tags/:org", a.updateAllTagsOrg},
		RouteData{"GET", "/generate/sha/:org/:repo/:sha", a.specificSha},
		RouteData{"GET", "/report/sha/:org/:repo/:sha", a.reportSha},
		RouteData{"GET", "/report/tag/repo/:org/:repo/:tag", a.reportTag},
		RouteData{"GET", "/report/tag/all/:tag", a.reportTagAll},
		RouteData{"GET", "/list/shas/:org/:repo", a.listShas},
		RouteData{"GET", "/list/tags/:org/:repo", a.listTagsRepo},
		RouteData{"GET", "/list/tags/:org", a.listTags},
		RouteData{"GET", "/list/projects", a.listProjects},
		RouteData{"GET", "/list/projects/:org", a.listProjectsOrg},
		RouteData{"GET", "/ui", a.formPath},
	})
	select {
	case err = <-server.Start(":" + port):
		done <- err
	case <-a.killChan:
		done <- errors.New(f.Format("was stopped: %s", server.Stop()))
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

	log.Println(git.Repository.FullName, git.AfterSha, git.Ref)
	c.String(200, "Thanks!")

	a.wrkr.AddTask(&git)
}

func (a *Application) updateAllTags(c *gin.Context) {
	if a.checkBack(c) {
		return
	}
	name := c.Param("repo")
	fullName := f.Format("%s/%s", c.Param("org"), name)
	dat, err := newTagsRunner(name, fullName).run()
	if err != nil {
		a.displayFailure(c, "Sorry, no can do. Problem: ["+err.Error()+"]")
		return
	}
	go func(dat map[string]string, name, fullName string) {
		for sha, ref := range dat {
			git := GitWebhook{
				Ref:      ref,
				AfterSha: sha,
				Repository: GitRepository{
					Name:     name,
					FullName: fullName,
				},
			}
			log.Println(fullName, sha, ref)
			a.wrkr.AddTask(&git)
		}
	}(dat, name, fullName)
	a.displaySuccess(c, "Yeah, I can do that. Check back in a minute")
}

func (a *Application) updateAllTagsOrg(c *gin.Context) {
	if a.checkBack(c) {
		return
	}
	org := c.Param("org")
	projects, err := a.rprtr.listProjectsByOrg(org)
	if err != nil {
		a.displayFailure(c, "Problemo: ["+err.Error()+"]")
		return
	}
	go func(projects []string) {
		for _, project := range projects {
			name := strings.SplitN(project, "/", 2)[1]
			dat, err := newTagsRunner(name, project).run()
			if err != nil {
				log.Println("[TAG UPDATER] Was unable to run tags against " + project + ": [" + err.Error() + "]")
				continue
			}
			go func(dat map[string]string, name string, project string) {
				for sha, ref := range dat {
					git := GitWebhook{
						Ref:      ref,
						AfterSha: sha,
						Repository: GitRepository{
							Name:     name,
							FullName: project,
						},
					}
					log.Println(project, sha, ref)
					a.wrkr.AddTask(&git)
				}
			}(dat, name, project)
		}
	}(projects)

	res := "Trying to run against:\n"
	for _, project := range projects {
		res += "\n" + project
	}

	a.displaySuccess(c, res)
}

func (a *Application) specificSha(c *gin.Context) {
	if a.checkBack(c) {
		return
	}
	name := c.Param("repo")
	fullName := f.Format("%s/%s", c.Param("org"), name)
	sha := c.Param("sha")
	code, _, _, err := nt.HTTP(nt.HEAD, f.Format("https://github.com/%s/commit/%s", fullName, sha), nt.NewHeaderBuilder().GetHeader(), nil)
	if err != nil {
		a.displayFailure(c, "could not verify this sha: "+err.Error())
		return
	}
	if code != 200 {
		a.displayFailure(c, f.Format("could not verify this sha, head code: %d", code))
		return
	}
	go func(name, fullName, sha string) {
		git := GitWebhook{
			AfterSha: sha,
			Repository: GitRepository{
				Name:     name,
				FullName: fullName,
			},
		}
		log.Println(fullName, sha)
		a.wrkr.AddTask(&git)
	}(name, fullName, sha)
	a.displaySuccess(c, "I got this, check back in a bit")
}

//

func (a *Application) reportSha(c *gin.Context) {
	if a.checkBack(c) {
		return
	}
	fullName := f.Format("%s/%s", c.Param("org"), c.Param("repo"))
	sha := c.Param("sha")
	deps, err := a.rprtr.reportBySha(fullName, sha)
	if err != nil {
		c.String(400, "Unable to do this: %s", err.Error())
		return
	}
	header := "Report for " + fullName + " at " + sha + "\n"
	t := table.NewTable(3, len(deps))
	for _, dep := range deps {
		t.Fill(dep.Name)
		t.Fill(dep.Version)
		t.Fill(dep.Language)
	}
	a.displaySuccess(c, header+t.SpaceColumn(1).NoBorders().Format().String())
}

//

func (a *Application) reportTag(c *gin.Context) {
	if a.checkBack(c) {
		return
	}
	tag := c.Param("tag")
	fullName := f.Format("%s/%s", c.Param("org"), c.Param("repo"))
	deps, err := a.rprtr.reportByTag2(c.Param("tag"), fullName)
	if err != nil {
		a.displayFailure(c, "Unable to do this: "+err.Error())
		return
	}
	header := "Report for " + fullName + " at " + tag + "\n"
	t := table.NewTable(3, len(deps))
	for _, dep := range deps {
		t.Fill(dep.Name)
		t.Fill(dep.Version)
		t.Fill(dep.Language)
	}
	a.displaySuccess(c, header+t.SpaceColumn(1).NoBorders().Format().String())
}
func (a *Application) reportTagAll(c *gin.Context) {
	if a.checkBack(c) {
		return
	}
	tag := c.Param("tag")
	deps, err := a.rprtr.reportByTag(tag)
	if err != nil {
		a.displayFailure(c, "Unable to do this: "+err.Error())
		return
	}
	res := ""
	for name, depss := range deps {
		res += name + " at " + tag
		t := table.NewTable(3, len(depss))
		for _, dep := range depss {
			t.Fill(dep.Name)
			t.Fill(dep.Version)
			t.Fill(dep.Language)
		}
		res += "\n" + t.NoBorders().SpaceColumn(1).Format().String() + "\n\n"
	}
	a.displaySuccess(c, res)
}

//

func (a *Application) listShas(c *gin.Context) {
	if a.checkBack(c) {
		return
	}
	fullName := f.Format("%s/%s", c.Param("org"), c.Param("repo"))
	shas, err := a.rprtr.listShas(fullName)
	if err != nil {
		a.displayFailure(c, err.Error())
		return
	}
	header := "List of Shas for " + fullName + "\n"
	t := table.NewTable(1, len(header))
	for _, sha := range shas {
		t.Fill(sha)
	}
	a.displaySuccess(c, header+t.NoBorders().Format().String())
}

//

func (a *Application) listTagsRepo(c *gin.Context) {
	if a.checkBack(c) {
		return
	}
	fullName := f.Format("%s/%s", c.Param("org"), c.Param("repo"))
	tags, err := a.rprtr.listTagsRepo(fullName)
	if err != nil {
		a.displayFailure(c, err.Error())
		return
	}
	header := "List of tags for " + fullName + "\n"
	t := table.NewTable(3, len(*tags))
	for k, v := range *tags {
		t.Fill(k)
		t.Fill("")
		t.Fill(v)
	}
	a.displaySuccess(c, header+t.SpaceColumn(1).NoBorders().Format().String())
}
func (a *Application) listTags(c *gin.Context) {
	if a.checkBack(c) {
		return
	}
	org := c.Param("org")
	tags, num, err := a.rprtr.listTags(org)
	if err != nil {
		a.displayFailure(c, err.Error())
		return
	}
	header := "List of tags for " + org + "\n"
	t := table.NewTable(2, num+len(*tags))
	for k, v := range *tags {
		t.Fill("")
		t.Fill("")
		t.Fill(k)
		for i, vv := range v {
			t.Fill(vv)
			if i != len(v)-1 {
				t.Fill(" ")
			}
		}
	}
	a.displaySuccess(c, header+t.SpaceColumn(1).NoBorders().Format().String())
}

func (a *Application) listProjects(c *gin.Context) {
	if a.checkBack(c) {
		return
	}
	ps, err := a.rprtr.listProjects()
	header := "List of projects\n"
	a.listProjectsWrk(ps, err, header, c)
}
func (a *Application) listProjectsOrg(c *gin.Context) {
	if a.checkBack(c) {
		return
	}
	org := c.Param("org")
	ps, err := a.rprtr.listProjectsByOrg(org)
	header := "List of projects for " + org + "\n"
	a.listProjectsWrk(ps, err, header, c)
}
func (a *Application) listProjectsWrk(ps []string, err error, header string, c *gin.Context) {
	if err != nil {
		a.displayFailure(c, err.Error())
		return
	}
	t := table.NewTable(1, len(ps))
	for _, v := range ps {
		t.Fill(v)
	}
	a.displaySuccess(c, header+t.NoBorders().Format().String())
}

//

func (a *Application) formPath(c *gin.Context) {
	var form Form
	if err := c.Bind(&form); err != nil {
		c.String(400, err.Error())
		return
	}
	if form.isEmpty() {
		ps, err := a.rprtr.listProjects()
		h := gin.H{}
		if err != nil {
			h["projects"] = "Sorry... could not\nload this."
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
		c.HTML(200, "form.html", h)
		return
	}
	buttonPress := form.findButtonPress()
	switch buttonPress {
	case ReportAllTag:
		c.Redirect(307, "/report/tag/all/"+form.ReportAllTag)
	case ReportTagSha:
		if form.ReportTag != "" {
			c.Redirect(307, f.Format("/report/tag/repo/%s/%s/%s", form.ReportOrg, form.ReportRepo, form.ReportTag))
		} else {
			c.Redirect(307, f.Format("/report/sha/%s/%s/%s", form.ReportOrg, form.ReportRepo, form.ReportSha))
		}
	case ListTags:
		if form.TagsRepo != "" {
			c.Redirect(307, f.Format("/list/tags/%s/%s", form.TagsOrg, form.TagsRepo))
		} else {
			c.Redirect(307, "/list/tags/"+form.TagsOrg)
		}
	case ListShas:
		c.Redirect(307, f.Format("/list/shas/%s/%s", form.ShasOrg, form.ShasRepo))
	case GenerateTag:
		if form.AllTagRepo != "" {
			c.Redirect(307, f.Format("/generate/tags/%s/%s", form.AllTagOrg, form.AllTagRepo))
		} else {
			c.Redirect(307, "/generate/tags/"+form.AllTagOrg)
		}
	case GenerateSha:
		c.Redirect(307, f.Format("/generate/sha/%s/%s/%s", form.ByShaOrg, form.ByShaRepo, form.ByShaSha))
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
