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
package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	com "github.com/venicegeo/vzutil-versioning/common"
	deps "github.com/venicegeo/vzutil-versioning/common/dependency"
	lan "github.com/venicegeo/vzutil-versioning/common/language"
)

var codeRE = regexp.MustCompile(`^(?:[0-9]|[a-z]|[A-Z]){4}$`)

func main() {
	var fileLocation string
	var listCode string
	var outFile string
	flag.StringVar(&fileLocation, "f", "", "File Location")
	flag.StringVar(&listCode, "c", "", "List Code")
	flag.StringVar(&outFile, "o", "", "Output File")
	flag.Parse()
	_, err := os.Stat(fileLocation)
	if err != nil {
		log.Fatalln(err)
	}
	if !codeRE.MatchString(listCode) {
		log.Fatalln("Bad code")
	}
	dat, err := ioutil.ReadFile(fileLocation)
	if err != nil {
		log.Fatalln(err)
	}
	ddeps, err := getDepsFromSoftwareList(dat, listCode)
	if err != nil {
		log.Fatalln(err)
	}
	pdeps := com.ProjectsDependencies{}
	for projectName, deps := range ddeps {
		pdeps[projectName] = com.ProjectDependencies{projectName, "", "", make([]string, len(deps), len(deps))}
		for i, dep := range deps {
			pdeps[projectName].Deps[i] = dep.String()
		}
		sort.Strings(pdeps[projectName].Deps)
	}
	dat, _ = json.MarshalIndent(pdeps, " ", "   ")
	if outFile != "" {
		ioutil.WriteFile(outFile, dat, 0644)
	} else {
		fmt.Println(string(dat))
	}
}

func getDepsFromSoftwareList(listDat []byte, indicesCode string) (map[string][]*deps.GenericDependency, error) {
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

	resultDepList := map[string][]*deps.GenericDependency{}
	unknownLangs := map[string]bool{}
	for _, record := range records {
		itemLanguage := lan.GetLanguage(record[indices[language]])
		if itemLanguage == lan.Unknown {
			unknownLangs[strings.TrimSuffix(record[indices[language]], "stack")] = true
			continue
		}
		components := strings.Split(strings.ToLower(record[indices[component]]), ",")
		for _, componentName := range components {
			componentName = strings.ToLower(strings.TrimSpace(componentName))
			if _, ok := resultDepList[componentName]; !ok {
				resultDepList[componentName] = []*deps.GenericDependency{}
			}
			resultDepList[componentName] = append(resultDepList[componentName], deps.NewGenericDependency(strings.ToLower(record[indices[name]]), strings.ToLower(record[indices[version]]), itemLanguage))
		}
	}
	for k, _ := range unknownLangs {
		fmt.Println("Software list contains unknown language:", k)

	}
	for _, list := range resultDepList {
		deps.RemoveExactDuplicates(&list)
	}
	return resultDepList, nil
}
