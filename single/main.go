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
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/venicegeo/vzutil-versioning/common"
	proj "github.com/venicegeo/vzutil-versioning/single/project"
	"github.com/venicegeo/vzutil-versioning/single/project/util"
)

func main() {
	var err error

	runInterruptHandler()

	//fmt.Println("### Generating direct dependencies...")

	var project *proj.Project

	if len(os.Args) != 3 {
		fmt.Println("The program arguments were incorrect. Usage: single [org/repo] [sha]")
		os.Exit(1)
	}

	name := strings.Split(os.Args[1], "/")[1]
	location, err := cloneAndCheckout(os.Args[1], os.Args[2], name)
	cleanup := func() { util.RunCommand("rm", "-rf", strings.TrimSuffix(location, name)) }
	defer cleanup()
	if err != nil {
		cleanup()
		fmt.Println("Error checking out:", err)
		os.Exit(1)
	}

	if project, err = proj.NewProject(fmt.Sprintf("%s/%s", location, name)); err != nil {
		cleanup()
		fmt.Println("Error creating project:", err)
		os.Exit(1)
	}
	if err = proj.Ingest(project, false); err != nil {
		cleanup()
		fmt.Println("Error ingesting project:", err)
		os.Exit(1)
	}

	//fmt.Printf("### Direct dependencies found for %s version %s\n", project.FolderName, project.Sha)

	ret := com.SingleReturn{project.FolderName, project.Sha, []string{}}
	for _, s := range project.GetDependencies() {
		ret.Deps = append(ret.Deps, s.FullString())
	}

	dat, err := json.MarshalIndent(ret, " ", "   ")
	if err != nil {
		cleanup()
		fmt.Println("Could not marshal return value:", err)
		os.Exit(1)
	}
	fmt.Println(string(dat))

}

func cloneAndCheckout(full_name, checkout, name string) (t string, err error) {
	t = fmt.Sprintf("%d", time.Now().UnixNano())
	var cmdRet util.CmdRet
	if cmdRet = util.RunCommand("mkdir", t); cmdRet.IsError() {
		return t, cmdRet.Error()
	}
	if t, err = filepath.Abs(t); err != nil {
		return t, err
	}
	rest := t
	t = fmt.Sprintf("%s/%s", t, name)
	if cmdRet = util.RunCommand("git", "clone", "https://github.com/"+full_name, t); cmdRet.IsError() {
		return t, cmdRet.Error()
	}
	if cmdRet = util.RunCommand("git", "-C", t, "checkout", checkout); cmdRet.IsError() {
		return t, cmdRet.Error()
	}
	return rest, nil
}

func runInterruptHandler() {
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		os.Exit(1)
	}()
}
