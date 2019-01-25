// Copyright 2018, RadiantBlue Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//sub
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package app

import (
	"fmt"
	"math"
	"strings"

	"github.com/gin-gonic/gin"
	h "github.com/venicegeo/vzutil-versioning/common/history"
	s "github.com/venicegeo/vzutil-versioning/web/app/structs"
	u "github.com/venicegeo/vzutil-versioning/web/util"
)

func (a *Application) projectConcept(c *gin.Context) {
	projId := c.Param("proj")
	project, err := a.rtrvr.GetProjectById(projId)
	if err != nil {
		c.String(400, "Error getting this project: %s", err.Error())
		return
	}

	alert := s.NewHtmlBasic("script", "")

	var form struct {
		Back       string `form:"button_back"`
		Util       string `form:"button_util"`
		Deployment string `form:"button_deployment"`
		Release    string `form:"button_release"`
		RepoUtil   string `form:"button_repository"`
		Repo       string `form:"button_repository_submit"`
	}
	if err := c.Bind(&form); err != nil {
		c.String(400, "Unable to bind form: %s", err.Error())
		return
	}

	if form.Back != "" {
		c.Redirect(303, "/ui")
		return
	} else if form.Util != "" {
		switch form.Util {
		case "Dependency Search":
			c.Redirect(303, "/depsearch/"+projId)
			return
		case "Delete Project":
			alert.SetValue(`alert("This has not been implemented yet");`)
		default:
			alert.SetValue(`alert("Unknown util method");`)
		}
	} else if form.Deployment != "" {
		alert.SetValue(`alert("This has not been implemented yet");`)
	} else if form.Release != "" {
		alert.SetValue(`alert("This has not been implemented yet");`)
	} else if form.RepoUtil != "" {
		switch form.RepoUtil {
		case "Add Repo":
			c.Redirect(303, "/addrepo/"+projId)
			return
		case "Generate All Tags":
			str, err := a.genTagsWrk(projId)
			if err != nil {
				alert.SetValue(u.Format(`alert("Unable to generate all tags: %s");`, err.Error()))
			} else {
				alert.SetValue(strings.Replace(u.Format(`alert("%s");`, str), "\n", `\n`, -1))
			}
		case "Reporting":
			c.Redirect(303, "/reportref/"+projId)
			return
		default:
			alert.SetValue(`alert("Unknown util method");`)
		}
	} else if form.Repo != "" {
		c.Redirect(303, "/repoTemp/"+projId+"/"+form.Repo) //TODO
		return
	}

	//special
	deploymentTable := s.NewHtmlTable().AddRow()
	addSpace := s.NewHtmlSubmitButton3("button_deployment", "Add Space", "button item green darker")
	deploymentTable.AddItem(0, addSpace)

	releaseTable := s.NewHtmlTable().AddRow()
	addRelease := s.NewHtmlSubmitButton3("button_release", "Add Release", "button item green darker")
	releaseTable.AddItem(0, addRelease)

	repos, err := project.GetAllRepositories()
	if err != nil {
		c.String(400, "Error getting repositories: %s", err.Error())
		return
	}
	repoTable := s.NewHtmlTable()
	for i, r := range repos {
		if i%3 == 0 {
			repoTable.AddRow()
		}
		repoTable.AddItem(i/3, s.NewHtmlButton(r.Fullname, "button_repository_submit", r.Id, "submit").Class("button item"))
	}
	repoTable.AddRow()
	addRepo := s.NewHtmlSubmitButton3("button_repository", "Add Repo", "button item green darker")
	genTags := s.NewHtmlSubmitButton3("button_repository", "Generate All Tags", "button item blue darker")
	reporting := s.NewHtmlSubmitButton3("button_repository", "Reporting", "button item yellow darker")
	repoTable.AddItem((len(repos)+2)/3, addRepo)
	repoTable.AddItem((len(repos)+2)/3, genTags)
	repoTable.AddItem((len(repos)+2)/3, reporting)
	//other
	c.HTML(200, "project_concept.html", gin.H{
		"deployments":  deploymentTable.Template(),
		"releases":     releaseTable.Template(),
		"repositories": repoTable.Template(),
		"alert":        alert.Template(),
	})
}

func (a *Application) repoTemp(c *gin.Context) {
	projId := c.Param("proj")
	repoId := c.Param("repo")
	repo, _, err := a.rtrvr.GetRepositoryById(repoId, projId)
	if err != nil {
		c.String(400, "Unable to find repository: %s", err.Error())
		return
	}
	refs, err := repo.GetAllRefs()
	if err != nil {
		c.String(500, "Unable to get repositories refs: %s", err.Error())
		return
	}

	tempAccord := s.NewHtmlAccordion()
	shas, _, err := repo.MapRefToShas()
	if err != nil {
		c.String(500, "Unable to map refs to shas: %s", err.Error())
		return
	}
	for _, ref := range refs {
		c := s.NewHtmlCollection()
		correctShas := shas["refs/"+ref]
		for i, sha := range correctShas {
			c.Add(s.NewHtmlSubmitButton2("button_sha", sha))
			if i < len(correctShas)-1 {
				c.Add(s.NewHtmlBr())
			}
		}
		tempAccord.AddItem(ref, s.NewHtmlForm(c).Post())
	}

	h := gin.H{}
	h["accordion"] = tempAccord.Sort().Template()
	c.HTML(200, "repo_temp.html", h)
}

func (a *Application) repoConcept(c *gin.Context) {
	var temp struct {
		Sha string `form:"sha"`
	}
	if err := c.Bind(&temp); err != nil {
		c.String(400, "Error binding form: %s", err.Error())
		return
	}

	type VNode struct {
		Id             string `json:"id"`
		Label          string `json:"label"`
		SecondaryLabel string `json:"labelSecondary"`
		Level          int    `json:"-"` //level
		Group          string `json:"group"`
		Xpos           int    `json:"x"`
		YPos           int    `json:"y"`
	}
	type VEdge struct {
		To     string `json:"to"`
		From   string `json:"from"`
		Arrows string `json:"arrows"`
		Hidden bool   `json:"hidden"`
	}
	nodes := []VNode{}
	edges := []VEdge{}
	tree, err := a.wrkr.History("venicegeo/vzutil-versioning")
	if err != nil {
		c.String(400, "Error creating history tree: %s\n", err.Error())
		return
	}

	leafs := tree.GetLeafs()
	sub := tree.GenerateSubtree(leafs, h.UP, 10)

	tree.ResetAllWeights(0)
	for _, root := range leafs {
		sub.TraverseFrom(root, h.UP, 1, func(node *h.HistoryNode, weight int) (bool, int) {
			if node.Weight != weight {
				node.Weight = weight
				for _, p := range node.Parents {
					edges = append(edges, VEdge{node.Sha, p, "to", false})
				}
				return true, weight
			}
			return false, weight
		})

	}
	tree.ResetAllWeights(-1)

	sub.ResetAllWeights(-1)
	max := -1
	for _, l := range leafs {
		temp := sub.CalculateHeights(l, h.UP, 0)
		if temp > max {
			max = temp
		}
	}

	sub.ReverseWeights(max)
	var tempNode VNode
	var tempName string
	var tempXOffset int
	knownBranches := map[string]int{}
	for _, n := range sub {
		//if n.IsStartOfBranch {
		tempName = strings.Join(n.Names, "|")
		//} else {
		//	tempName = ""
		//}
		if _, ok := knownBranches[tempName]; !ok {
			knownBranches[tempName] = len(knownBranches)
		}
		tempXOffset = knownBranches[tempName]
		if !n.IsStartOfBranch {
			tempName = ""
		}
		tempNode = VNode{n.Sha, n.Sha[:7], tempName, n.Weight, "good", tempXOffset * 200, n.Weight * -150}
		nodes = append(nodes, tempNode)
	}
	//Include tags on side - WIP
	if false {
		missing := sub.FindMissingNames(tree)
		fmt.Println(missing)
		fmt.Println(len(missing))
		missingSquare := int(math.Ceil(math.Sqrt(float64(len(missing)))))
		var missingLevel int
		for i, sha := range missing {
			if i%missingSquare == 0 {
				missingLevel = max
			} else {
				missingLevel--
			}
			fmt.Println(sha[:7], missingLevel)
			node := tree[sha]
			nodes = append(nodes, VNode{sha, sha[:7], strings.Join(node.Names, "\n"), missingLevel, "warning", 0, 0})
			if i%missingSquare != 0 {
				edges = append(edges, VEdge{sha, missing[i-1], "to", true})
			}
		}
	}

	c.HTML(200, "repo_overview_concept.html", gin.H{"nodes": nodes, "edges": edges})
}
