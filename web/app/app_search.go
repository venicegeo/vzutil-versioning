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
	"bytes"
	"encoding/json"
	"strings"

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
	h := gin.H{
		"data":             "Search Results will appear here",
		"depsearchname":    tmp.DepName,
		"depsearchversion": tmp.DepVersion,
	}
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

	rawDat, err := es.GetAll(a.index, "dependency", u.Format(`
{
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
}`, depName, depVersion))
	if err != nil {
		return 500, "Error querying database: " + err.Error()
	}

	containingDeps := make([]string, len(rawDat), len(rawDat))
	buf := bytes.NewBufferString("Searching for:\n")
	for i, b := range rawDat {
		containingDeps[i] = u.Format(`"%s"`, b.Id)
		var dep es.Dependency
		if err = json.Unmarshal(b.Dat, &dep); err != nil {
			buf.WriteString(u.Format("\tError decoding %s\n", b.Id))
		} else {
			buf.WriteString("\t")
			buf.WriteString(dep.String())
			buf.WriteString("\n")
		}
	}
	buf.WriteString("\n\n\n")

	q := u.Format(`
{
	"terms":{
		"dependencies":[%s]
	}
}`, strings.Join(containingDeps, ","))
	s := `{
	"timestamp":"desc"
}`
	rawDat, err = es.GetAll(a.index, "repository_entry", q, s)
	if err != nil {
		return 500, "Unable to query repos: " + err.Error()
	}

	test := map[string]map[string][]string{}
	for _, b := range rawDat {
		var entry es.RepositoryEntry
		if err = json.Unmarshal(b.Dat, &entry); err != nil {
			return 500, "Error getting entry: " + err.Error()
		}
		if _, ok := test[entry.RepositoryFullName]; !ok {
			test[entry.RepositoryFullName] = map[string][]string{}
		}
		if _, ok := test[entry.RepositoryFullName][entry.RefName]; !ok {
			test[entry.RepositoryFullName][entry.RefName] = []string{}
		}
		test[entry.RepositoryFullName][entry.RefName] = append(test[entry.RepositoryFullName][entry.RefName], entry.Sha)
	}
	for repoName, refs := range test {
		buf.WriteString(repoName)
		buf.WriteString("\n")
		for refName, shas := range refs {
			buf.WriteString(u.Format("\t%s\n", refName))
			for _, sha := range shas {
				buf.WriteString(u.Format("\t\t %s \n", sha))
			}
		}
	}

	return 200, buf.String()
}
