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
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	proj "github.com/venicegeo/vzutil-versioning/single/project"
)

func main() {
	var err error

	runInterruptHandler()

	fmt.Println("### Generating direct dependencies...")

	var project *proj.Project

	if len(os.Args) != 3 {
		log.Fatalln("Not enough args")
	}

	name := strings.Split(os.Args[1], "/")[1]
	location, err := cloneAndCheckout(os.Args[1], os.Args[2], name)
	cleanup := func() { exec.Command("rm", "-rf", location).Run() }
	defer cleanup()
	if err != nil {
		cleanup()
		log.Fatalln("checking out:", err)
	}

	if project, err = proj.NewProject(fmt.Sprintf("%s/%s", location, name)); err != nil {
		cleanup()
		log.Fatalln("creating project:", err)
	}
	if err = proj.Ingest(project, false); err != nil {
		cleanup()
		log.Fatalln("ingesting:", err)
	}

	fmt.Printf("### Direct dependencies found for %s version %s\n", project.FolderName, project.Sha)
	for _, s := range project.GetDependencies() {
		fmt.Printf("###   %s\n", s.FullString())
	}

}

func cloneAndCheckout(full_name, checkout, name string) (t string, err error) {
	t = fmt.Sprintf("%d", time.Now().Unix())
	if err = exec.Command("mkdir", t).Run(); err != nil {
		return t, err
	}
	if t, err = filepath.Abs(t); err != nil {
		return t, err
	}
	rest := t
	t = fmt.Sprintf("%s/%s", t, name)
	var dat []byte
	if dat, err = exec.Command("git", "clone", "https://github.com/"+full_name, t).Output(); err != nil {
		log.Println("clone:", string(dat))
		return t, err
	}
	cmd := exec.Command("git", "-C", t, "checkout", checkout)
	if dat, err = cmd.Output(); err != nil {
		log.Println("checkout:", string(dat))
		return t, err
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
