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
	"encoding/json"

	"github.com/gin-gonic/gin"
	"github.com/venicegeo/vzutil-versioning/web/es"
	u "github.com/venicegeo/vzutil-versioning/web/util"
)

func (a *Application) uiSearchForDep(c *gin.Context) {
	type Search struct {
		Ui           string `form:"button_back"`
		DepName      string `form:"depsearchname"`
		DepVersion   string `form:"depsearchversion"`
		ButtonSearch string `form:"button_depsearch"`
	}
	var tmp Search
	if err := c.Bind(&tmp); err != nil {
		return
	}
	h := gin.H{"data": "Search Results will appear here"}
	if tmp.Ui != "" {
		c.Redirect(307, "/ui")
	} else if tmp.ButtonSearch != "" {
		code, dat := a.searchForDepWrk(tmp.DepName, tmp.DepVersion)
		h["data"] = dat
		c.HTML(code, "depsearch.html", h)
	} else {
		c.HTML(200, "depsearch.html", h)
	}
}

func (a *Application) searchForDep(c *gin.Context) {
	depName := c.Param("dep")
	depVersion := c.Param("version")
	c.String(a.searchForDepWrk(depName, depVersion))
}

func (a *Application) searchForDepWrk(depName, depVersion string) (int, string) {
	tmp := ""
	query := u.Format(`
{
	"size": %d,
	"query":{
		"bool":{
		"must":[
			{
				"term":{
					"name":"%s"
				}
			},{
				"wildcard":{
					"version":"%s*"
				}
			}
		]
		}
	}
}`, a.searchSize, depName, depVersion)
	resp, err := a.index.SearchByJSON("dependency", query)
	if err != nil {
		return 500, "Error querying database: " + err.Error()
	}
	hits := resp.GetHits()
	deps := make([]es.Dependency, resp.NumHits(), resp.NumHits())
	for i, hit := range *hits {
		var dep es.Dependency
		if err = json.Unmarshal(*hit.Source, &dep); err != nil {
			return 500, "Error unmarshalling dependency: " + err.Error()
		}
		deps[i] = dep
	}
	{
		tmp = "Searching for:\n"
		for _, dep := range deps {
			tmp += u.Format("\t%s\n", dep.String())
		}
		tmp += "\n\n\n"
	}

	query = u.Format(`{
	"size":%d,
	"query":{
		"bool":{
			"should":[`, a.searchSize)
	for i, dep := range deps {
		query += u.Format(`{
			"term":{
				"refs.entries.dependencies":"%s"
			}
		}`, dep.GetHashSum())
		if i != len(deps)-1 {
			query += ","
		}
	}
	query += `]
		}
	}
}
`

	projects, err := es.HitsToProjects(a.index.SearchByJSON("project", query))
	if err != nil {
		return 500, "Error getting projects: " + err.Error()
	}
	//projectEntries := map[string]*es.ProjectEntries{}
	//						 projectName   ref   shas
	containingProjects := map[string]map[string][]string{}
	for _, project := range *projects {
		for _, ref := range project.Refs {
			for _, entry := range ref.Entries {
				breakk := false
				for _, dep := range entry.Dependencies {
					for _, toSearch := range deps {
						if toSearch.GetHashSum() == dep {
							breakk = true
							if _, ok := containingProjects[project.FullName]; !ok {
								containingProjects[project.FullName] = map[string][]string{ref.Name: []string{entry.Sha}}
							} else {
								containingProjects[project.FullName][ref.Name] = append(containingProjects[project.FullName][ref.Name], entry.Sha)
							}
						}
						if breakk {
							break
						}
					}
					if breakk {
						break
					}
				}
			}
		}
	}
	for projectName, e1 := range containingProjects {
		tmp += projectName + "\n"
		for refName, e2 := range e1 {
			tmp += "\t" + refName + "\n"
			for _, sha := range e2 {
				tmp += "\t\t" + sha + "\n"
			}
		}
	}
	return 200, tmp
}
