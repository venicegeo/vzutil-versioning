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
package ingest

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/venicegeo/vzutil-versioning/common/dependency"
	lan "github.com/venicegeo/vzutil-versioning/common/language"
	"github.com/venicegeo/vzutil-versioning/single/project/issue"
)

var removeNewLineRE = regexp.MustCompile(`\r?\n`)
var removeInfoSpace = regexp.MustCompile(`(\[INFO\] +)`)

type PomProjectWrapper struct {
	Project              *PomProject `json:"project"`
	Parent               *PomProjectWrapper
	Children             []*PomProjectWrapper
	dependencyManagement []*Item
	ProjectWrapper
}

func (p *PomProjectWrapper) compileCheck() {
	var _ IProjectWrapper = (*PomProjectWrapper)(nil)
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

func (c *PomCollection) GetResults() (total dependency.GenericDependencies, issues []*issue.Issue, err error) {
	for _, pom := range *c {
		if pom.Parent != nil {
			continue
		}
		deps, iss, err := getResultsAndFromChildren(pom, true, []*Item{}, nil)
		if err != nil {
			return nil, nil, err
		}
		issues = append(issues, iss...)
		total.Add(deps...)
	}
	return total, issues, nil
}
func getResultsAndFromChildren(pom *PomProjectWrapper, isRoot bool, dependencyManagement []*Item, previousMvnDeps []*MvnDependency) (dependency.GenericDependencies, []*issue.Issue, error) {
	deps, issues, err := pom.GetResults()
	if err != nil {
		return nil, issues, err
	}
	mvnDeps, mvnError := pom.generateMvnDependencies()
	if mvnError != nil && isRoot {
		return nil, nil, mvnError
	} else if mvnError != nil {
		issues = append(issues, issue.NewIssue("Failed to build [%s] with maven", pom.Project.ArtifactId))
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
		deps.Add(childDeps...)
		issues = append(issues, childIssues...)
	}
	return deps, issues, nil
}

func (pw *PomProjectWrapper) GetResults() (dependency.GenericDependencies, []*issue.Issue, error) {
	if err := pw.replaceVariables(); err != nil {
		return nil, pw.issues, err
	}
	dependenciesMap := pw.Project.Dependencies
	var dependencies []*Item
	if val, ok := dependenciesMap["dependency"]; !ok {
	} else {
		dat, err := json.Marshal(val)
		if err != nil {
			return nil, pw.issues, err
		}
		var rep *Item
		var reps []*Item
		err1 := json.Unmarshal(dat, &rep)
		err2 := json.Unmarshal(dat, &reps)
		if err1 != nil && err2 != nil {
			return nil, pw.issues, err1
		} else if err1 == nil {
			dependencies = []*Item{rep}
		} else if err2 == nil {
			dependencies = reps
		} else {
			return nil, pw.issues, errors.New("Bad logic")
		}
	}
	var deps dependency.GenericDependencies
	for _, dep := range dependencies {
		deps.Add(dependency.NewGenericDependency(dep.ArtifactId, dep.Version, pw.name, lan.Java))
	}
	dependencyManagerMap := pw.Project.DependencyManagement
	if len(dependencyManagerMap) > 0 {
		var dependencyReference []*Item
		if depensi, ok := dependencyManagerMap["dependencies"]; ok {
			if depens, ok := depensi.(map[string]interface{}); ok {
				if val, ok := depens["dependency"]; !ok {
				} else {
					dat, err := json.Marshal(val)
					if err != nil {
						return nil, pw.issues, err
					}
					var rep *Item
					var reps []*Item
					err1 := json.Unmarshal(dat, &rep)
					err2 := json.Unmarshal(dat, &reps)
					if err1 != nil && err2 != nil {
						return nil, pw.issues, err1
					} else if err1 == nil {
						dependencyReference = []*Item{rep}
					} else if err2 == nil {
						dependencyReference = reps
					} else {
						return nil, pw.issues, errors.New("Bad logic")
					}
				}
			}
		}
		pw.dependencyManagement = dependencyReference
	}
	return deps, pw.issues, nil
}

func (p *PomProjectWrapper) compareAndReplaceDependecies(deps dependency.GenericDependencies, mvnDeps []*MvnDependency, dependencyManagement []*Item) {
	if deps == nil || mvnDeps == nil {
		return
	}
	for _, pomDep := range deps {
		for _, manDep := range dependencyManagement {
			if pomDep.GetName() == manDep.ArtifactId && pomDep.GetVersion() != manDep.Version {
				p.addIssue(issue.NewVersionMismatch(pomDep.GetName(), pomDep.GetVersion(), manDep.Version))
				pomDep.SetVersion(manDep.Version)
			}
		}
		for _, mvnDep := range mvnDeps {
			if pomDep.GetName() == mvnDep.ArtifactId && pomDep.GetVersion() != mvnDep.Version {
				p.addIssue(issue.NewVersionMismatch(pomDep.GetName(), pomDep.GetVersion(), mvnDep.Version))
				pomDep.SetVersion(mvnDep.Version)
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
			p.addIssue(issue.NewUnusedVariable(k, v))
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

func (p *PomProjectWrapper) generateMvnDependencies() ([]*MvnDependency, error) {
	cmdRet := GenerateMvnReport(p.location)
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
		return []*MvnDependency{}, nil
	}
	lines = lines[start:stop]
	for lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	dependencies := []*MvnDependency{}
	for _, line := range lines {
		parts := strings.Split(line, ":")
		dependencies = append(dependencies, &MvnDependency{
			GroupId:    parts[0],
			ArtifactId: parts[1],
			Packaging:  parts[2],
			Version:    parts[3],
		})
	}
	return dependencies, nil
}
