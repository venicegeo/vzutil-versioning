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
	"bufio"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	deps "github.com/venicegeo/vzutil-versioning/plural/project/dependency"
	lan "github.com/venicegeo/vzutil-versioning/plural/project/language"
	sha "github.com/venicegeo/vzutil-versioning/plural/project/shastore"
	"github.com/venicegeo/vzutil-versioning/plural/project/states"
	"github.com/venicegeo/vzutil-versioning/plural/project/util"

	"gopkg.in/yaml.v2"
)

type Config struct {
	AboutYmlCloneUrl        string                 `json:"about_clone_url"`
	AboutYmlBranch          string                 `json:"about_branch"`
	PathToAbout             string                 `json:"about_path"`
	SoftwareListCloneUrl    string                 `json:"software_list_clone_url"`
	SoftwareListBranch      string                 `json:"software_list_branch"`
	PathToList              string                 `json:"list_path"`
	ListIndicesCode         string                 `json:"list_indices_code"`
	PathToShaStore          string                 `json:"sha_store_path"`
	ProjectCloneUrl         string                 `json:"projects_clone_url"`
	Projects                map[string]ProjectInfo `json:"projects"`
	ProjectsExtended        map[string]ProjectInfo `json:"projects_extended"`
	DepExecptions           []string               `json:"deps_exception_regexs"`
	RepoCheckPathExceptions []string               `json:"repo_check_exceptions"`
	Bundles                 map[string][]string    `json:"package_bundles"`
}

type ConfigResults struct {
	ProjectName             string
	AboutDepList            deps.GenericDependencies
	SoftwareDepList         deps.GenericDependencies
	ShaStore                sha.ShaStore
	Projects                *Projects
	ProjectsExtended        *Projects
	DepExceptions           []string
	TargetFolder            string
	RepoCheckPathExceptions []string
	Bundles                 map[string][]string
}

func GetConfigs() ([]string, error) {
	files, err := ioutil.ReadDir("./")
	if err != nil {
		return nil, err
	}
	re := regexp.MustCompile("^config(?:_([^.]+))*.json$")
	res := []string{}
	for _, f := range files {
		if re.MatchString(f.Name()) {
			res = append(res, f.Name())
		}
	}
	return res, nil
}

func RunConfig(fileName string) (*ConfigResults, error) {
	projectName := re.FindStringSubmatch(fileName)[1]
	if fileName == "" {
		return nil, doesntExist()
	}
	dat, err := ioutil.ReadFile(fileName)
	if err != nil {
		return nil, err
	}
	var config Config
	if err = json.Unmarshal(dat, &config); err != nil {
		return nil, err
	}
	now := time.Now().Unix()
	folderName := fmt.Sprintf("%d/", now)
	if err = util.RunCommand("mkdir", folderName); err != nil {
		return nil, err
	}
	state.ComparingAbout = config.AboutYmlCloneUrl != "" && state.CloneLists
	state.ComparingSoftwareList = config.SoftwareListCloneUrl != "" && state.CloneLists
	isShaStore := config.PathToShaStore != ""
	defer os.RemoveAll(folderName)
	if state.ComparingAbout {
		aboutNameA := strings.Split(strings.Replace(config.AboutYmlCloneUrl, ".git", "", -1), "/")
		aboutName := aboutNameA[len(aboutNameA)-1]
		fmt.Println("Cloning about yaml...")
		if config.AboutYmlBranch != "" {
			if err = util.RunCommand("git", "clone", "-b", config.AboutYmlBranch, config.AboutYmlCloneUrl, folderName+aboutName); err != nil {
				return nil, err
			}
		} else if err = util.RunCommand("git", "clone", config.AboutYmlCloneUrl, folderName+aboutName); err != nil {
			return nil, err
		}
	}
	if state.ComparingSoftwareList {
		listNameA := strings.Split(strings.Replace(config.SoftwareListCloneUrl, ".git", "", -1), "/")
		listName := listNameA[len(listNameA)-1]
		fmt.Println("Cloning deps list...")
		if config.SoftwareListBranch != "" {
			if err = util.RunCommand("git", "clone", "-b", config.SoftwareListBranch, config.SoftwareListCloneUrl, folderName+listName); err != nil {
				return nil, err
			}
		} else if err = util.RunCommand("git", "clone", config.SoftwareListCloneUrl, folderName+listName); err != nil {
			return nil, err
		}
	}
	var aboutDat, listDat, shaStoreDat []byte = nil, nil, nil
	var aboutDepList, softwareDepList deps.GenericDependencies = nil, nil
	var shaStore sha.ShaStore = nil
	if state.ComparingAbout {
		if aboutDat, err = ioutil.ReadFile(folderName + config.PathToAbout); err != nil {
			return nil, err
		}
		if aboutDepList, err = getDepsFromAboutYaml(aboutDat); err != nil {
			return nil, err
		}
	}
	if state.ComparingSoftwareList {
		if listDat, err = ioutil.ReadFile(folderName + config.PathToList); err != nil {
			return nil, err
		}
		if softwareDepList, err = getDepsFromSoftwareList(listDat, config.ListIndicesCode); err != nil {
			return nil, err
		}
	}
	if state.ComparingSoftwareList && isShaStore {
		if shaStoreDat, err = ioutil.ReadFile(folderName + config.PathToShaStore); err != nil {
			return nil, err
		}
		if shaStore, err = getShaStore(shaStoreDat); err != nil {
			return nil, err
		}
	}
	targetFolder := fmt.Sprintf("%d/", now)
	projects := Projects{}
	extendedProjects := Projects{}
	for k, v := range config.Projects {
		v.fix(k, config.ProjectCloneUrl)
		if projects[k], err = NewProject(k, v, targetFolder); err != nil {
			return nil, err
		}
	}
	for k, v := range config.ProjectsExtended {
		v.fix(k, config.ProjectCloneUrl)
		if extendedProjects[k], err = NewProject(k, v, targetFolder); err != nil {
			return nil, err
		}
	}
	for i, v := range config.RepoCheckPathExceptions {
		config.RepoCheckPathExceptions[i] = targetFolder + v
	}
	return &ConfigResults{projectName, aboutDepList, softwareDepList, shaStore,
		&projects, &extendedProjects, config.DepExecptions, targetFolder, config.RepoCheckPathExceptions, config.Bundles}, nil
}

func doesntExist() error {
	errMaking := func(err error) error {
		return errors.New("Config file not found, error making file: " + err.Error())
	}

	file, err := os.Create("config_project.json")
	if err != nil {
		return errMaking(err)
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	defer writer.Flush()
	dat, err := json.MarshalIndent(Config{
		AboutYmlCloneUrl:     "github.com/ORG/PROJECT",
		PathToAbout:          "PROJECT/path/to/about",
		SoftwareListCloneUrl: "github.com/ORG/PROJECT2",
		PathToShaStore:       "PROJECT2/path/to/sha",
		PathToList:           "PROJECT2/path/to/list",
		ProjectCloneUrl:      "github.com/ORG/",
		Projects:             map[string]ProjectInfo{"repo_name": ProjectInfo{ComponentName: "project_name"}},
		DepExecptions:        []string{`^github.com\/ORG\/.+$`},
		Bundles:              map[string][]string{"package": []string{"sub_packageA", "sub_packageB"}}}, " ", "   ")
	if err != nil {
		return errMaking(err)
	}
	if _, err = writer.WriteString(string(dat)); err != nil {
		return errMaking(err)
	}

	return errors.New("Config file not found. A default config was generated")
}

func getDepsFromAboutYaml(aboutDat []byte) (resultDepList deps.GenericDependencies, err error) {
	yml := map[string]interface{}{}
	if err = yaml.Unmarshal(aboutDat, &yml); err != nil {
		return nil, err
	}
	re := regexp.MustCompile(`^[^:]+:[^:]+(:[^:]*)$`)
	stacksByLanguage, simpleStack := map[interface{}]interface{}{}, []interface{}{}
	ok := false
	if _, state.AboutByLanguage = yml["stacks"]; state.AboutByLanguage {
		if stacksByLanguage, ok = yml["stacks"].(map[interface{}]interface{}); !ok {
			return nil, errors.New("Stacks type not map[interface{}]interface{}")
		}
	}
	if _, state.AboutSimple = yml["stack"]; state.AboutSimple {
		if simpleStack, ok = yml["stack"].([]interface{}); !ok {
			return nil, errors.New("Stack type not []interface{}")
		}
	}
	if state.AboutByLanguage {
		for stackNameI, stackDepListI := range stacksByLanguage {
			stackName, ok := stackNameI.(string)
			if !ok {
				return nil, fmt.Errorf("Stack name [%v] not string", stackNameI)
			}
			stackDepList, ok := stackDepListI.([]interface{})
			if !ok {
				return nil, errors.New("Dependency list not []interface{}")
			}
			stackLanguage := lan.GetLanguage(stackName)
			if stackLanguage == lan.Unknown {
				fmt.Println("About yaml contains unknown language:", strings.TrimSuffix(stackName, "stack"))
				continue
			}
			for _, stackDepI := range stackDepList {
				stackDep, ok := stackDepI.(string)
				if !ok {
					return nil, fmt.Errorf("Dep [%v] not string", stackDepI)
				}
				if re.MatchString(stackDep) {
					stackDep = strings.TrimSuffix(stackDep, re.FindStringSubmatch(stackDep)[1])
				}
				resultDepList.Add(deps.NewGenericDependencyStr(stackDep + "::" + string(stackLanguage)))
			}
		}
		resultDepList.RemoveExactDuplicates()
	} else if state.AboutSimple {
		for _, idep := range simpleStack {
			dep, ok := idep.(string)
			if !ok {
				return nil, fmt.Errorf("Dep [%v] not string", idep)
			}
			resultDepList.Add(deps.NewGenericDependencyStr(dep))
		}
	}
	return resultDepList, err
}

func getDepsFromSoftwareList(listDat []byte, indicesCode string) (deps.GenericDependencies, error) {
	name, version, component, language := 0, 1, 2, 3
	indices := [4]int64{}
	for i := range indices {
		tempi, err := strconv.ParseInt(indicesCode[i:i+1], 36, 64)
		if err != nil {
			return nil, err
		}
		indices[i] = tempi
	}
	reader := csv.NewReader(bytes.NewReader(listDat))
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(records) > 0 {
		records = records[1:]
	}

	resultDepList := deps.GenericDependencies{}
	unknownLangs := map[string]bool{}
	for _, record := range records {
		itemLanguage := lan.GetLanguage(record[indices[language]])
		if itemLanguage == lan.Unknown {
			unknownLangs[strings.TrimSuffix(record[indices[language]], "stack")] = true
			continue
		}
		components := strings.Split(strings.ToLower(record[indices[component]]), ",")
		for _, componentName := range components {
			componentName = strings.TrimSpace(componentName)
			resultDepList.Add(deps.NewGenericDependency(record[indices[name]], record[indices[version]], componentName, itemLanguage))
		}
	}
	for k, _ := range unknownLangs {
		fmt.Println("Software list contains unknown language:", k)

	}
	resultDepList.RemoveExactDuplicates()
	return resultDepList, nil
}

func getShaStore(storeDat []byte) (sha.ShaStore, error) {
	store := sha.ShaStore{}
	reader := csv.NewReader(bytes.NewReader(storeDat))
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	for _, record := range records {
		store = append(store, &sha.ShaStoreEntry{record[0], record[1], record[2]})
	}
	return store, nil
}
