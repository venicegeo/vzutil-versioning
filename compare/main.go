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
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"regexp"
	"sort"
	"strings"

	com "github.com/venicegeo/vzutil-versioning/common"
	deps "github.com/venicegeo/vzutil-versioning/common/dependency"
	"github.com/venicegeo/vzutil-versioning/common/table"
)

var re = regexp.MustCompile(``)

func readFile(filename string) (com.ProjectsDependencies, error) {
	var fileDat []byte
	var err error
	if fileDat, err = ioutil.ReadFile(filename); err != nil {
		log.Fatalln(err)
	}
	var fileDeps com.ProjectsDependencies
	err = json.Unmarshal(fileDat, &fileDeps)
	return fileDeps, err
}

type CompareStruct struct {
	ActualName      string
	ExpectedName    string
	ActualDeps      []string
	ExpectedDeps    []string
	ExpectedExtra   []string
	ExpectedMissing []string
	Agreed          []string
}

func NewCompareStruct(actualName, expectedName string) *CompareStruct {
	return &CompareStruct{actualName, expectedName, []string{}, []string{}, []string{}, []string{}, []string{}}
}

func main() {
	var file1, file2, outFile, string1, string2 string
	flag.StringVar(&file1, "a", "", "Actual File")
	flag.StringVar(&file2, "e", "", "Expected File")
	flag.StringVar(&outFile, "o", "", "Output File")
	flag.StringVar(&string1, "as", "", "Actual String")
	flag.StringVar(&string2, "es", "", "Expected String")
	flag.Parse()

	var expected, actual com.ProjectsDependencies
	var err error

	if file1 == "" && string1 == "" {
		log.Fatalln("Either the actual file or string must be provided.")
	} else if file1 != "" && string1 != "" {
		log.Fatalln("Only one actual source can be provided.")
	} else if file1 != "" {
		if actual, err = readFile(file1); err != nil {
			log.Fatalln(err)
		}
	} else {
		string1 = strings.TrimPrefix(strings.TrimSuffix(string1, `'`), `'`)
		if err = json.Unmarshal([]byte(string1), &actual); err != nil {
			log.Fatalln(err)
		}
	}

	if file2 == "" && string2 == "" {
		log.Fatalln("Either the expected file or string must be provided.")
	} else if file2 != "" && string2 != "" {
		log.Fatalln("Only one expected source can be provided.")
	} else if file2 != "" {
		if expected, err = readFile(file2); err != nil {
			log.Fatalln(err)
		}
	} else {
		string2 = strings.TrimPrefix(strings.TrimSuffix(string2, `'`), `'`)
		if err = json.Unmarshal([]byte(string2), &expected); err != nil {
			log.Fatalln(err)
		}
	}

	compares := []*CompareStruct{}
	for projectName, project := range actual {
		var maxSim float64 = 0.0
		var temp float64 = 0.0
		var maxKey string = ""
		for k2, _ := range expected {
			temp = similarity(projectName, k2)
			if temp >= 0.5 && maxSim < temp {
				maxSim = temp
				maxKey = k2
			}
		}
		str := NewCompareStruct(projectName, maxKey)
		for _, s := range project.Deps {
			str.ActualDeps = append(str.ActualDeps, deps.NewGenericDependencyStr(s).String())
		}
		if str.ExpectedName != "" {
			if proj, ok := expected[str.ExpectedName]; ok {
				for _, s := range proj.Deps {
					str.ExpectedDeps = append(str.ExpectedDeps, deps.NewGenericDependencyStr(s).String())
				}
			}
			delete(expected, str.ExpectedName)
		}
		compares = append(compares, str)
	}
	for projectName, project := range expected {
		str := NewCompareStruct("", projectName)
		for _, s := range project.Deps {
			str.ExpectedDeps = append(str.ExpectedDeps, deps.NewGenericDependencyStr(s).String())
		}
		compares = append(compares, str)
	}

	done := make(chan bool, len(compares))
	searchList := func(a, b, found, notfound *[]string) {
		var f = false
		for _, dep := range *a {
			f = false
			for _, exp := range *b {
				if dep == exp {
					f = true
					if found != nil {
						*found = append(*found, dep)
					}
					break
				}
			}
			if !f {
				*notfound = append(*notfound, dep)
			}
		}
	}
	findDifs := func(cmp *CompareStruct) {
		searchList(&cmp.ActualDeps, &cmp.ExpectedDeps, &cmp.Agreed, &cmp.ExpectedMissing)
		searchList(&cmp.ExpectedDeps, &cmp.ActualDeps, nil, &cmp.ExpectedExtra)
		cmp.ExpectedMissing, cmp.ExpectedExtra = similaritySort(cmp.ExpectedMissing, cmp.ExpectedExtra)
		sort.Strings(cmp.Agreed)
		done <- true
	}
	for _, cmp := range compares {
		go findDifs(cmp)
	}
	for i := 0; i < len(compares); i++ {
		<-done
	}

	output := ""
	for _, cmp := range compares {
		m := max(len(cmp.Agreed), len(cmp.ExpectedMissing), len(cmp.ExpectedExtra))
		if m == 0 {
			continue
		}
		t := table.NewTable(3, m+1)
		agreed := ""
		missing := ""
		extra := ""
		output += fmt.Sprintf("Comparing actual in [%s] to list [%s]\n", cmp.ActualName, cmp.ExpectedName)
		t.Fill("Agreed", "Missing in List", "Extra in List")
		for i := 0; i < m; i++ {
			agreed = ""
			missing = ""
			extra = ""
			if len(cmp.Agreed) > i {
				agreed = cmp.Agreed[i]
			}
			if len(cmp.ExpectedMissing) > i {
				missing = cmp.ExpectedMissing[i]
			}
			if len(cmp.ExpectedExtra) > i {
				extra = cmp.ExpectedExtra[i]
			}
			t.Fill(agreed, missing, extra)
		}
		output += t.SpaceAllColumns().NoRowBorders().Format().String()
		output += "\n\n\n\n"
	}
	if outFile == "" {
		fmt.Println(output)
	} else {
		ioutil.WriteFile(outFile, []byte(output), 0644)
	}
}

func max(a ...int) int {
	if len(a) == 0 {
		return 0
	}
	max := a[0]
	for _, b := range a {
		if b > max {
			max = b
		}
	}
	return max
}
