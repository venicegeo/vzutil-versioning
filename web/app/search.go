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

func (a *Application) searchForDep(c *gin.Context) {
	depName := c.Param("dep")
	depVersion := c.Param("version")
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
		c.String(500, "Error querying database:", err.Error())
		return
	}
	hits := resp.GetHits()
	deps := make([]es.Dependency, resp.NumHits(), resp.NumHits())
	for i, hit := range *hits {
		var dep es.Dependency
		if err = json.Unmarshal(*hit.Source, &dep); err != nil {
			c.String(500, "Error unmarshalling dependency:", err)
		}
		deps[i] = dep
	}
	{
		tmp := "Searching for:\n"
		for _, dep := range deps {
			tmp += u.Format("\t%s\n", dep.String())
		}
		c.String(200, tmp+"\n\n\n")
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
		c.String(500, "Error getting projects:", err)
		return
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
						break
					}
					if breakk {
						break
					}
				}
			}
		}
	}
	tmp := ""
	for projectName, e1 := range containingProjects {
		tmp += projectName + "\n"
		for refName, e2 := range e1 {
			tmp += "\t" + refName + "\n"
			for _, sha := range e2 {
				tmp += "\t\t" + sha + "\n"
			}
		}
	}
	c.String(200, tmp)
}
