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
	"os"
	"os/exec"
	"sync"

	"github.com/venicegeo/vzutil-versioning/common"
)

func main() {
	if _, err := os.Stat("single"); err != nil {
		log.Fatalln(err)
	}
	var configLocation string
	var outFile string
	flag.StringVar(&configLocation, "c", "", "Config File")
	flag.StringVar(&outFile, "o", "", "Output File")
	flag.Parse()
	if _, err := os.Stat(configLocation); err != nil {
		log.Fatalln(err)
	}
	dat, err := ioutil.ReadFile(configLocation)
	if err != nil {
		log.Fatalln(err)
	}
	var projects map[string]string
	if err = json.Unmarshal(dat, &projects); err != nil {
		log.Fatalln(err)
	}
	projectData := com.ProjectsDependencies{}
	mux := sync.Mutex{}
	wg := sync.WaitGroup{}
	wg.Add(len(projects))
	for project, branch := range projects {
		go func(project, branch string) {
			if branch == "" {
				branch = "master"
			}
			cmd := exec.Command("./single", project, branch)
			if dat, err = cmd.Output(); err != nil {
				log.Fatalln(cmd.Args, err)
			}
			var ret com.ProjectDependencies
			if err = json.Unmarshal(dat, &ret); err != nil {
				log.Fatalln(err)
			}
			mux.Lock()
			projectData[ret.Name] = ret
			wg.Done()
			mux.Unlock()
		}(project, branch)
	}
	wg.Wait()
	if dat, err = json.MarshalIndent(projectData, " ", "   "); err != nil {
		log.Fatalln(err)
	}
	if outFile != "" {
		ioutil.WriteFile(outFile, dat, 0644)
	} else {
		fmt.Println(string(dat))
	}
}
