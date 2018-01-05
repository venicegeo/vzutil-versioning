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
	"os/signal"
	"syscall"

	proj "github.com/venicegeo/vzutil-versioning/single/project"
)

func main() {
	runInterruptHandler()

	fmt.Println("### Generating direct dependencies...")

	var err error
	var project *proj.Project

	if project, err = proj.NewProject("../bf-api"); err != nil {
		log.Fatalln(err)
	}
	if err = proj.Ingest(project, false); err != nil {
		log.Fatalln(err)
	}

	fmt.Printf("### Direct dependencies found for %s version %s\n", project.FolderName, project.Sha)
	for _, s := range project.GetDependencies() {
		fmt.Printf("###   %s\n", s)
	}
}

func runInterruptHandler() {
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		os.Exit(1)
	}()
}
