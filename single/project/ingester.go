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
package project

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/venicegeo/vzutil-versioning/common/dependency"
	lan "github.com/venicegeo/vzutil-versioning/common/language"
	"github.com/venicegeo/vzutil-versioning/single/project/ingest"
	"github.com/venicegeo/vzutil-versioning/single/project/issue"

	"gopkg.in/yaml.v2"
)

type Ingester struct {
	fileReader IFileReader
}

var re = regexp.MustCompile(`([^\/]+$)`)

func Ingest(project *Project, prnt bool) (err error) {
	i := Ingester{&FileReader{}}
	if err = project.findDepFiles(); err != nil {
		return err
	}
	if prnt {
		str := "Ingesting " + project.FolderName
		for _, loc := range project.DepLocations {
			str += "\n  - " + loc
		}
		fmt.Println(str)
	}
	if errs := i.IngestProject(project); len(errs) != 0 {
		errStr := errs[0].Error()
		for i := 1; i < len(errs); i++ {
			errStr += "\n" + errs[i].Error()
		}
		return fmt.Errorf("%s:%s", project.FolderName, errStr)
	}
	return nil
}

func (i *Ingester) IngestProject(p *Project) (errors []error) {
	var deps, tempDeps []*dependency.GenericDependency
	var issues, tempIssues []*issue.Issue
	var err error
	javaHit := false
	for _, filePath := range p.DepLocations {
		fileName := re.FindStringSubmatch(filePath)[0]
		switch lan.FileToLang[fileName] {
		case lan.Java:
			if !javaHit {
				tempDeps, issues, err = i.ingestJavaProject(p)
				javaHit = true
			}
		case lan.JavaScript:
			tempDeps, issues, err = i.ingestJavaScriptFile(filePath, p)
		case lan.Go:
			tempDeps, issues, err = i.ingestGoFile(filePath, p)
		case lan.Python:
			tempDeps, issues, err = i.ingestPythonFile(filePath, p)
		}
		if tempDeps != nil {
			deps = append(deps, tempDeps...)
		}
		if tempIssues != nil {
			issues = append(issues, tempIssues...)
		}
		if err != nil {
			errors = append(errors, err)
		}
	}
	dependency.RemoveExactDuplicates(&deps)
	p.Dependencies = deps
	p.AddIssue(issues...)
	return errors
}

func (i *Ingester) ingestJavaProject(p *Project) ([]*dependency.GenericDependency, []*issue.Issue, error) {
	poms := ingest.PomCollection{}
	for _, filePath := range p.DepLocations {
		if !strings.HasSuffix(filePath, "pom.xml") {
			continue
		}
		data, err := i.fileReader.Read(filePath)
		if err != nil {
			return nil, nil, err
		}
		jsn, err := XmlToMap(data)
		if err != nil {
			return nil, nil, err
		}
		if _, ok := jsn["project"]; ok {
			if jproj, ok := jsn["project"].(map[string]interface{}); ok {
				for k, v := range map[string]reflect.Kind{"dependencies": reflect.Interface, "repositories": reflect.Interface, "properties": reflect.String, "dependencyManagement": reflect.Interface} {
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
		var projectWrapper ingest.PomProjectWrapper
		if err = json.Unmarshal(data, &projectWrapper); err != nil {
			fmt.Println(string(data))
			return nil, nil, fmt.Errorf("ingestJavaProject %s unmarshal: %s", filePath, err.Error())
		}
		fileName := re.FindStringSubmatch(filePath)[0]
		projectWrapper.SetProperties(strings.TrimSuffix(filePath, fileName), p.FolderName)
		poms.Add(&projectWrapper)
	}
	poms.BuildHierarchy(false)

	//poms.PrintHierarchy()
	return poms.GetResults()
}

func (i *Ingester) ingestJavaScriptFile(filePath string, p *Project) ([]*dependency.GenericDependency, []*issue.Issue, error) {
	data, err := i.fileReader.Read(filePath)
	if err != nil {
		return nil, nil, err
	}
	var projectWrapper ingest.JsProjectWrapper
	if err = json.Unmarshal(data, &projectWrapper); err != nil {
		return nil, nil, err
	}
	projectWrapper.SetProperties(p.FolderLocation, p.FolderName)
	return projectWrapper.GetResults()
}

func (i *Ingester) ingestGoFile(filePath string, p *Project) ([]*dependency.GenericDependency, []*issue.Issue, error) {
	var yamlData, lockData []byte
	var yml ingest.GlideYaml
	var lock ingest.GlideLock
	var err error

	if yamlData, err = i.fileReader.Read(filePath); err != nil {
		return nil, nil, err
	}
	if err = yaml.Unmarshal(yamlData, &yml); err != nil {
		return nil, nil, err
	}

	if lockData, err = i.fileReader.Read(strings.TrimSuffix(filePath, "glide.yaml") + "glide.lock"); err != nil {
		lockData = []byte("")
	}
	if err = yaml.Unmarshal(lockData, &lock); err != nil {
		return nil, nil, err
	}
	projectWrapper := ingest.GoProjectWrapper{Yaml: &yml, Lock: &lock}
	projectWrapper.SetProperties(p.FolderLocation, p.FolderName)
	return projectWrapper.GetResults()
}

func (i *Ingester) ingestPythonFile(filePath string, p *Project) ([]*dependency.GenericDependency, []*issue.Issue, error) {
	isPip := strings.HasSuffix(filePath, "requirements.txt")
	reqDat, err := i.fileReader.Read(filePath)
	if err != nil {
		return nil, nil, err
	}
	if isPip {
		devDat, err := i.fileReader.Read(strings.TrimSuffix(filePath, "requirements.txt") + "requirements-dev.txt")
		if err != nil {
			devDat = []byte("")
		}
		projectWrapper := ingest.PipProjectWrapper{Filedat: reqDat, DevFileDat: devDat}
		projectWrapper.SetProperties(p.FolderLocation, p.FolderName)
		return projectWrapper.GetResults()
	} else {
		projectWrapper := ingest.CondaProjectWrapper{Filedat: reqDat}
		projectWrapper.SetProperties(p.FolderLocation, p.FolderName)
		return projectWrapper.GetResults()
	}
}
