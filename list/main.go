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
	byProj := map[string][]string{}
	for _, dep := range ddeps {
		if _, ok := byProj[dep.GetProject()]; !ok {
			byProj[dep.GetProject()] = []string{dep.FullString()}
		} else {
			byProj[dep.GetProject()] = append(byProj[dep.GetProject()], dep.FullString())
		}
	}
	for _, d := range byProj {
		sort.Strings(d)
	}
	dat, _ = json.MarshalIndent(byProj, " ", "   ")
	if outFile != "" {
		ioutil.WriteFile(outFile, dat, 0644)
	} else {
		fmt.Println(string(dat))
	}
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
