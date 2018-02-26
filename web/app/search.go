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
			"refs.entries.dependencies":"%s"
		}`, dep)
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
	//	comeBackTo := map[string][][]string{}
	//	projectTagShas := map[string][]es.TagSha{}
	//	breakk := false
	//	for _, project := range *projects {
	//		entries, err := project.GetEntries()
	//		if err != nil {
	//			c.String(500, "Error getting project entries")
	//			return
	//		}
	//		tagShas, err := project.GetTagShas()
	//		if err != nil {
	//			c.String(500, "Error getting project tag shas")
	//			return
	//		}
	//		projectTagShas[project.FullName] = tagShas
	//		projectEntries[project.FullName] = entries
	//		for sha, entry := range *entries {
	//			if entry.EntryReference != "" {
	//				if _, ok := comeBackTo[project.FullName]; !ok {
	//					comeBackTo[project.FullName] = [][]string{}
	//				}
	//				comeBackTo[project.FullName] = append(comeBackTo[project.FullName], []string{entry.EntryReference, sha})
	//				continue
	//			}
	//			breakk = false
	//			for _, dep := range entry.Dependencies {
	//				for _, baddep := range deps {
	//					if dep == baddep.GetHashSum() {
	//						if _, ok := containingProjects[project.FullName]; !ok {
	//							containingProjects[project.FullName] = []string{}
	//						}
	//						containingProjects[project.FullName] = append(containingProjects[project.FullName], sha)
	//						breakk = true
	//					}
	//					if breakk {
	//						break
	//					}
	//				}
	//				if breakk {
	//					break
	//				}
	//			}
	//		}
	//	}
	//	for project, entries := range comeBackTo {
	//		containing, ok := containingProjects[project]
	//		if !ok {
	//			continue
	//		}
	//		for _, entry := range entries {
	//			reference := entry[0]
	//			sha := entry[1]
	//			for _, containingsha := range containing {
	//				if containingsha == reference {
	//					containingProjects[project] = append(containingProjects[project], sha)
	//					break
	//				}
	//			}
	//		}
	//	}
	//	{
	//		tmp := ""
	//		for projectName, shas := range containingProjects {
	//			tmp += projectName + "\n"
	//			tagShas := projectTagShas[projectName]
	//			for _, sha := range shas {
	//				tmp += "\t" + sha
	//				if tagShas != nil {
	//					for tag, shaa := range *tagShas {
	//						if shaa == sha {
	//							tmp += " " + tag
	//						}
	//					}
	//				}
	//				tmp += "\n"
	//			}
	//			tmp += "\n"
	//		}
	//		c.String(200, tmp)
	//}
}
