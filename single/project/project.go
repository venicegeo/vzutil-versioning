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
	"path/filepath"
	"regexp"
	"strings"

	"github.com/venicegeo/vzutil-versioning/single/project/dependency"
	"github.com/venicegeo/vzutil-versioning/single/project/issue"
	lan "github.com/venicegeo/vzutil-versioning/single/project/language"
	"github.com/venicegeo/vzutil-versioning/single/project/util"
)

type Project struct {
	FolderName     string
	FolderLocation string
	Sha            string
	Dependencies   dependency.GenericDependencies
	DepLocations   []string
	Issues         []*issue.Issue `json:"issues,omitempty"`
}

var folderNameRE = regexp.MustCompile(`\/?([^\/]+)\/?$`)

func NewProject(folderLocation string) (proj *Project, err error) {
	folderName := folderNameRE.FindStringSubmatch(folderLocation)[1]
	if folderLocation, err = filepath.Abs(folderLocation); err != nil {
		return nil, err
	}
	var currentDir string
	var cmdRet util.CmdRet
	if currentDir, err = os.Getwd(); err != nil {
		return nil, err
	}
	if err = os.Chdir(folderLocation); err != nil {
		return nil, err
	}
	if cmdRet = util.RunCommand("git", "rev-parse", "HEAD"); cmdRet.IsError() {
		return nil, cmdRet.Error()
	}
	if err = os.Chdir(currentDir); err != nil {
		return nil, err
	}
	proj = &Project{FolderName: folderName, FolderLocation: folderLocation, Sha: strings.TrimSpace(cmdRet.Stdout)}
	return proj, nil
}

func (p *Project) GetDependencies() dependency.GenericDependencies {
	return p.Dependencies
}

func (p *Project) findDepFiles() (err error) {
	visit := func(path string, f os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if f.IsDir() {
			return nil
		}
		if util.IsVendorPath(path, p.FolderLocation) || util.IsDotGitPath(path, p.FolderLocation) {
			return nil
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
