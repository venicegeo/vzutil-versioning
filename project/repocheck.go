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
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/venicegeo/vzutil-versioning/project/issue"
	"github.com/venicegeo/vzutil-versioning/project/states"
	"github.com/venicegeo/vzutil-versioning/project/util"
)

func RepoCheck(projects *Projects, pathExceptions []string) error {
	checkChan := make(chan error, len(*projects))
	combErr := ""
	copyrightFileExtentions := []string{
		".go",
		".java",
		".js",
		".py",
		".ts",
		".tsx",
	}
	copyrightStatements := []*regexp.Regexp{
		regexp.MustCompile(`Copyright 201[6789], RadiantBlue Technologies, Inc\.`),
		regexp.MustCompile(`WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND`),
		regexp.MustCompile(`Apache License, Version 2\.0`),
	}
	reReadMe := regexp.MustCompile(`^README(?:(?:.txt)|(?:.md))?$`)
	reLiscence := regexp.MustCompile(`^LICENSE(?:(?:.txt)|(?:.md))?$`)

	check := func(p *Project) {
		folder, err := os.Open(p.FolderLocation)
		if err != nil {
			checkChan <- err
			return
		}
		files, err := folder.Readdirnames(0)
		if err != nil {
			checkChan <- err
			return
		}
		hasReadme, hasLiscence, hasAbout := false, false, false
		for _, f := range files {
			if reReadMe.MatchString(f) {
				hasReadme = true
			} else if reLiscence.MatchString(f) {
				hasLiscence = true
			} else if f == `.about.yml` {
				hasAbout = true
			}
		}
		if !hasReadme {
			p.AddIssue(issue.NewIssue("Missing README"))
		}
		if !hasLiscence {
			p.AddIssue(issue.NewIssue("Missing LISCENCE"))
		}
		if !hasAbout {
			p.AddIssue(issue.NewIssue("Missing .about"))
		}

		visit := func(path string, f os.FileInfo, err error) error {
			if f.IsDir() {
				return nil
			}
			if util.IsVendorPath(path, p.FolderLocation) || util.IsDotGitPath(path, p.FolderLocation) {
				return nil
			}
			for _, exception := range pathExceptions {
				//TODO not perfect, like many things
				if strings.Contains(path, exception) {
					return nil
				}
			}
			correctType := false
			for _, v := range copyrightFileExtentions {
				if strings.HasSuffix(path, v) {
					correctType = true
					break
				}
			}
			if !correctType {
				return nil
			}
			dat, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}
			str := string(dat)
			for _, statement := range copyrightStatements {
				if !statement.MatchString(str) {
					p.AddIssue(issue.NewIssue("File [%s] does not contain copyright statement [%s]", path, statement))
				}
			}
			return nil
		}
		if err := filepath.Walk(p.FolderLocation, visit); err != nil {
			checkChan <- err
			return
		}
		checkChan <- nil
	}
	for _, p := range *projects {
		if state.Async {
			go check(p)
		} else {
			check(p)
		}
	}
	for j := 0; j < len(*projects); j++ {
		err := <-checkChan
		if err != nil {
			combErr += err.Error() + "\n"
		}
	}
	if combErr != "" {
		return fmt.Errorf("%s", combErr)
	}
	return nil
}
