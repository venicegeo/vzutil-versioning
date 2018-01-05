/*
Copyright 2017, RadiantBlue Technologies, Inc.

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
package project

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/venicegeo/vzutil-versioning/plural/project/dependency"
	"github.com/venicegeo/vzutil-versioning/plural/project/issue"
	lan "github.com/venicegeo/vzutil-versioning/plural/project/language"
	"github.com/venicegeo/vzutil-versioning/plural/project/util"
)

type Projects map[string]*Project

func (p *Projects) GetAllDependencies() dependency.GenericDependencies {
	res := dependency.GenericDependencies{}
	for _, proj := range *p {
		res = append(res, proj.GetDependencies()...)
	}
	return res
}

type Project struct {
	ProjectInfo
	repoName       string
	FolderLocation string
	Dependencies   dependency.GenericDependencies
	DepLocations   []string
	Issues         []*issue.Issue `json:"issues,omitempty"`
}

type ProjectInfo struct {
	CloneUrl      string   `json:"cloneUrl"`
	CloneBranch   string   `json:"cloneBranch"`
	ComponentName string   `json:"componentName"`
	WalkIgnore    []string `json:"walkIgnore"`
}

func (pi *ProjectInfo) fix(repoName, cloneUrl string) {
	if strings.TrimSpace(pi.ComponentName) == "" {
		pi.ComponentName = repoName
	}
	if strings.TrimSpace(pi.CloneUrl) == "" {
		pi.CloneUrl = fixLocation(cloneUrl) + repoName
	}
}

func NewProject(repoName string, projectInfo ProjectInfo, clonedLocation string) (*Project, error) {
	proj := Project{ProjectInfo: projectInfo, repoName: repoName}
	return &proj, proj.SetLocation(clonedLocation)
}

func (p *Project) GetDependencies() dependency.GenericDependencies {
	return p.Dependencies
}

func (p *Project) SetLocation(location string) (err error) {
	location, err = filepath.Abs(fixLocation(location) + p.repoName)
	p.FolderLocation = location + "/"
	return err
}

func fixLocation(location string) string {
	if !strings.HasSuffix(location, "/") && strings.TrimSpace(location) != "" {
		location = location + "/"
	}
	return location
}

func (p *Project) CloneAndMove(cloneChan chan<- error) {
	branch := strings.TrimSpace(p.CloneBranch)
	if branch == "" {
		branch = "master"
	}
	cloneChan <- exec.Command("git", "clone", "-b", branch, p.CloneUrl, p.FolderLocation).Run()
}

func (p *Project) findDepFiles() (err error) {
	visit := func(path string, f os.FileInfo, err error) error {
		if f.IsDir() {
			return nil
		}
		if util.IsVendorPath(path, p.FolderLocation) || util.IsDotGitPath(path, p.FolderLocation) {
			return nil
		}
		for _, ignore := range p.WalkIgnore {
			if strings.HasPrefix(path, p.FolderLocation+fixLocation(ignore)) {
				return nil
			}
		}
		if _, ok := lan.FileToLang[f.Name()]; ok {
			p.DepLocations = append(p.DepLocations, path)
		}
		return nil
	}
	return filepath.Walk(p.FolderLocation, visit)
}

func (p *Project) AddIssue(issue ...*issue.Issue) {
	p.Issues = append(p.Issues, issue...)
}
