/*
Copyright 2018, RadiantBlue Technologies, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package resolve

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strings"

	d "github.com/venicegeo/vzutil-versioning/common/dependency"
	i "github.com/venicegeo/vzutil-versioning/common/issue"
	lan "github.com/venicegeo/vzutil-versioning/common/language"
	"github.com/venicegeo/vzutil-versioning/single/resolve/mvn"
	"github.com/venicegeo/vzutil-versioning/single/util"
)

//TODO fix this array stuff

var removeNewLineRE = regexp.MustCompile(`\r?\n`)
var removeInfoSpace = regexp.MustCompile(`(\[INFO\] +)`)
var getFilePath = regexp.MustCompile(`([^\/]+$)`)

func (r *Resolver) ResolvePomXml(location string, test bool) (d.Dependencies, i.Issues, error) {
	poms := PomCollection{}
	data, err := r.readFile(location)
	if err != nil {
		return nil, nil, err
	}
	jsn, err := util.XmlToMap(data)
	if err != nil {
		return nil, nil, err
	}
	if _, ok := jsn["project"]; ok {
		if jproj, ok := jsn["project"].(map[string]interface{}); ok {
			for k, v := range map[string]reflect.Kind{"dependencies": reflect.Interface, "repositories": reflect.Interface, "properties": reflect.String, "dependencyManagement": reflect.Interface, "build": reflect.Interface, "profiles": reflect.Interface} {
				if keyName, ok := jproj[k]; ok {
					if reflect.TypeOf(keyName).Kind() != reflect.MapOf(reflect.TypeOf(""), reflect.TypeOf(v)).Kind() {
						jproj[k] = reflect.New(reflect.MapOf(reflect.TypeOf(""), reflect.TypeOf(v))).Interface()
					}
				}
			}
		}
	}
	data, err = json.MarshalIndent(jsn, " ", "   ")
	if err != nil {
		return nil, nil, err
	}
	var projectWrapper PomProjectWrapper
	if err = json.Unmarshal(data, &projectWrapper); err != nil {
		return nil, nil, fmt.Errorf("ingestJavaProject %s unmarshal: %s", location, err.Error())
	}
	fileName := getFilePath.FindStringSubmatch(location)[0]
	projectWrapper.SetProperties(strings.TrimSuffix(location, fileName), "")
	poms.Add(&projectWrapper)

	poms.BuildHierarchy(false)

	//poms.PrintHierarchy()
	if deps, issues, err := poms.GetResults(); err != nil {
		return deps, issues, err
	} else {
		sort.Sort(deps)
		sort.Sort(issues)
		return deps, issues, nil
	}
}

type PomProjectWrapper struct {
	Project              *PomProject `json:"project"`
	Parent               *PomProjectWrapper
	Children             []*PomProjectWrapper
	dependencyManagement []*Item
	ProjectWrapper
}

func (p *PomProjectWrapper) AddChild(p2 *PomProjectWrapper) {
	p.Children = append(p.Children, p2)
	p2.Parent = p
}

type PomCollection []*PomProjectWrapper

func (c *PomCollection) Add(pom *PomProjectWrapper) {
	*c = append(*c, pom)
}
func (c *PomCollection) BuildHierarchy(debug bool) {
	for _, pom := range *c {
		if debug {
			fmt.Println("Looking at pom:", pom.Project.ArtifactId)
		}
		parentInfo := pom.Project.Parent
		if parentInfo == nil {
			if debug {
				fmt.Println("  Has no parent")
			}
			continue
		}
		for _, pom2 := range *c {
			pom2parentInfo := pom2.Project.Parent
			if (pom2.Project.GroupId == parentInfo.GroupId || pom2parentInfo != nil && pom2parentInfo.GroupId == parentInfo.GroupId) && pom2.Project.ArtifactId == parentInfo.ArtifactId {
				if debug {
					fmt.Println("  I found my parent:", pom2.Project.ArtifactId)
				}
				pom2.AddChild(pom)
			}
		}
	}
}
func (c *PomCollection) PrintHierarchy() {
	for _, pom := range *c {
		if pom.Parent == nil {
			fmt.Println(pom.Project.ArtifactId)
			pom.printChildren("  ")
		}
	}
}
func (pom *PomProjectWrapper) printChildren(space string) {
	for _, child := range pom.Children {
		fmt.Println(space + child.Project.ArtifactId)
		child.printChildren(space + "  ")
	}
}

type PomProject struct {
	ModelVersion string `json:"modelVersion"`
	Item
	Packaging            string                 `json:"packaging"`
	Parent               *Item                  `json:"parent"`
	Repositories         map[string]interface{} `json:"repositories"`
	Dependencies         map[string]interface{} `json:"dependencies"`
	DependencyManagement map[string]interface{} `json:"dependencyManagement"`
	Properties           map[string]string      `json:"properties"`
	Build                map[string]interface{} `json:"build"`
	Profiles             map[string]interface{} `json:"profiles"`
}

type PomBuild struct {
	Plugins map[string]interface{} `json:"plugins"`
}
type PomProfile struct {
}

type PomRepositoryArr struct {
	Repository []Repository `json:"repository"`
}
type Repository struct {
	Id   string `json:"id"`
	Name string `json:"name"`
	Url  string `json:"url"`
}
type PomDependencyArr struct {
	Dependency []Item `json:"dependency"`
}
type Item struct {
	GroupId    string `json:"groupId"`
	ArtifactId string `json:"artifactId"`
	Version    string `json:"version,omitempty"`
	Scope      string `json:"scope,omitempty"`
}

func (c *PomCollection) GetResults() (total d.Dependencies, issues i.Issues, err error) {
	for _, pom := range *c {
		if pom.Parent != nil {
			continue
		}
		deps, iss, err := getResultsAndFromChildren(pom, true, []*Item{}, nil)
		if err != nil {
			return nil, nil, err
		}
		issues = append(issues, iss...)
		total = append(total, deps...)
	}
	for _, dep := range total {
		if dep.Version == "" {
			issues = append(issues, i.NewMissingVersion(dep.Name))
		}
	}
	return total, issues, nil
}
func getResultsAndFromChildren(pom *PomProjectWrapper, isRoot bool, dependencyManagement []*Item, previousMvnDeps []*mvn.MvnDependency) (d.Dependencies, i.Issues, error) {
	deps, issues, err := pom.GetResults()
	if err != nil {
		return nil, issues, err
	}
	mvnDeps, mvnError := pom.generateMvnDependencies()
	/*if mvnError != nil && isRoot {
		return nil, nil, mvnError
	} else*/if mvnError != nil {
		issues = append(issues, i.NewIssue("Failed to build [%s] with maven", pom.Project.ArtifactId))
		mvnDeps = previousMvnDeps
	}
	pom.compareAndReplaceDependecies(deps, mvnDeps, dependencyManagement)
	for _, child := range pom.Children {
		temp := make([]*Item, len(dependencyManagement))
		copy(temp, dependencyManagement)
		toSend := append(temp, pom.dependencyManagement...)
		childDeps, childIssues, err := getResultsAndFromChildren(child, false, toSend, mvnDeps)
		if err != nil {
			return deps, issues, err
		}
		deps = append(deps, childDeps...)
		issues = append(issues, childIssues...)
	}
	return deps, issues, nil
}

func (pw *PomProjectWrapper) GetResults() (d.Dependencies, i.Issues, error) {
	if err := pw.replaceVariables(); err != nil {
		return nil, pw.issues, err
	}

	getItems := func(mapp map[string]interface{}, key string) ([]*Item, error) {
		var items []*Item
		if val, ok := mapp[key]; !ok {
		} else {
			dat, err := json.Marshal(val)
			if err != nil {
				return nil, err
			}
			var rep *Item
			var reps []*Item
			err1 := json.Unmarshal(dat, &rep)
			err2 := json.Unmarshal(dat, &reps)
			if err1 != nil && err2 != nil {
				return nil, err1
			} else if err1 == nil {
				items = []*Item{rep}
			} else if err2 == nil {
				items = reps
			} else {
				return nil, errors.New("Bad logic")
			}
		}
		return items, nil
	}

	dependencies, err := getItems(pw.Project.Dependencies, "dependency")
	if err != nil {
		return nil, pw.issues, err
	}
	dependencyManagerMap := pw.Project.DependencyManagement
	if _, ok := dependencyManagerMap["dependencies"]; ok {
		if _, ok = dependencyManagerMap["dependencies"].(map[string]interface{}); ok {
			dependencyReference, err := getItems(dependencyManagerMap["dependencies"].(map[string]interface{}), "dependency")
			if err != nil {
				return nil, pw.issues, err
			}
			pw.dependencyManagement = dependencyReference
		}
	}
	buildMap := pw.Project.Build
	if _, ok := buildMap["plugins"]; ok {
		if _, ok = buildMap["plugins"].(map[string]interface{}); ok {
			plugins, err := getItems(buildMap["plugins"].(map[string]interface{}), "plugin")
			if err != nil {
				return nil, pw.issues, err
			}
			dependencies = append(dependencies, plugins...)
		}
	}
	if pw.Project.Parent != nil {
		dependencies = append(dependencies, pw.Project.Parent)
	}
	profilesMap := pw.Project.Profiles
	if _, ok := profilesMap["profile"]; ok {
		if _, ok = profilesMap["profile"].(map[string]interface{}); ok {
			if _, ok = profilesMap["profile"].(map[string]interface{})["build"]; ok {
				if _, ok = profilesMap["profile"].(map[string]interface{})["build"].(map[string]interface{}); ok {
					if _, ok = profilesMap["profile"].(map[string]interface{})["build"].(map[string]interface{})["plugins"]; ok {
						if _, ok = profilesMap["profile"].(map[string]interface{})["build"].(map[string]interface{})["plugins"].(map[string]interface{}); ok {
							plugins, err := getItems(profilesMap["profile"].(map[string]interface{})["build"].(map[string]interface{})["plugins"].(map[string]interface{}), "plugin")
							if err != nil {
								return nil, pw.issues, nil
							}
							dependencies = append(dependencies, plugins...)
						}
					}
				}
			}
			if _, ok = profilesMap["profile"].(map[string]interface{})["dependencies"]; ok {
				if _, ok = profilesMap["profile"].(map[string]interface{})["dependencies"].(map[string]interface{}); ok {
					deps, err := getItems(profilesMap["profile"].(map[string]interface{})["dependencies"].(map[string]interface{}), "dependency")
					if err != nil {
						return nil, pw.issues, nil
					}
					dependencies = append(dependencies, deps...)
				}
			}
		}
	}

	deps := make(d.Dependencies, len(dependencies), len(dependencies))
	for i, dep := range dependencies {
		deps[i] = d.NewDependency(dep.ArtifactId, dep.Version, lan.Java)
	}
	return deps, pw.issues, nil
}

func (p *PomProjectWrapper) compareAndReplaceDependecies(deps d.Dependencies, mvnDeps []*mvn.MvnDependency, dependencyManagement []*Item) {
	if deps == nil || mvnDeps == nil {
		return
	}
	for index, pomDep := range deps {
		for _, manDep := range dependencyManagement {
			if pomDep.Name == manDep.ArtifactId && pomDep.Version != manDep.Version {
				p.issues = append(p.issues, i.NewVersionMismatch(pomDep.Name, pomDep.Version, manDep.Version))
				pomDep.Version = manDep.Version
				deps[index] = pomDep
			}
		}
		for _, mvnDep := range mvnDeps {
			if pomDep.Name == mvnDep.ArtifactId && pomDep.Version != mvnDep.Version {
				p.issues = append(p.issues, i.NewVersionMismatch(pomDep.Name, pomDep.Version, mvnDep.Version))
				pomDep.Version = mvnDep.Version
				deps[index] = pomDep
			}
		}
	}
}

func (p *PomProjectWrapper) getParentAndMyVars(pass map[string]string) map[string]string {
	if p.Parent != nil {
		pass = p.Parent.getParentAndMyVars(pass)
		for k, v := range p.Parent.Project.Properties {
			pass[k] = v
		}
	}
	for k, v := range p.Project.Properties {
		pass[k] = v
	}
	return pass
}

func (p *PomProjectWrapper) replaceVariables() error {
	vars := p.getParentAndMyVars(map[string]string{})
	data, err := json.MarshalIndent(p.Project, " ", "   ")
	if err != nil {
		return err
	}
	str := string(data)
	for k, v := range vars {
		replace := fmt.Sprintf("${%s}", k)
		if !strings.Contains(str, replace) && k != "java.version" {
			p.issues = append(p.issues, i.NewUnusedVariable(k, v))
		}
		str = strings.Replace(str, replace, v, -1)
	}
	newProject := &PomProject{}
	if err = json.Unmarshal([]byte(str), newProject); err != nil {
		return err
	}
	p.Project = newProject
	return nil
}

func (p *PomProjectWrapper) generateMvnDependencies() ([]*mvn.MvnDependency, error) {
	cmdRet := mvn.GenerateMvnReport(p.location)
	if cmdRet.IsError() {
		return nil, fmt.Errorf("Unable to generate maven report at %s\n%s", p.location, cmdRet.String())
	}
	data := cmdRet.Stdout
	if !strings.Contains(data, "BUILD SUCCESS") {
		return nil, errors.New("Maven build failure. Check authentication\n" + data)
	}
	lines := strings.Split(data, "\n")
	{
		format := []string{}
		for _, line := range lines {
			temp := removeNewLineRE.ReplaceAllString(line, "")
			temp = removeInfoSpace.ReplaceAllString(line, "")
			format = append(format, temp)
		}
		lines = format
	}
	start, stop := -1, -1
	for k, line := range lines {
		if line == "The following files have been resolved:" {
			start = k + 1
		}
		if start != -1 && line == "------------------------------------------------------------------------" {
			stop = k - 1
			break
		}
	}
	if start == -1 {
		return []*mvn.MvnDependency{}, nil
	}
	lines = lines[start:stop]
	for lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	dependencies := []*mvn.MvnDependency{}
	for _, line := range lines {
		parts := strings.Split(line, ":")
		dependencies = append(dependencies, &mvn.MvnDependency{
			GroupId:    parts[0],
			ArtifactId: parts[1],
			Packaging:  parts[2],
			Version:    parts[3],
		})
	}
	return dependencies, nil
}
