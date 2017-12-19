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
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	proj "github.com/venicegeo/vzutil-versioning/project"
	deps "github.com/venicegeo/vzutil-versioning/project/dependency"
	"github.com/venicegeo/vzutil-versioning/project/reporting"
	"github.com/venicegeo/vzutil-versioning/project/states"
	"github.com/venicegeo/vzutil-versioning/project/util"
)

var projects *proj.Projects
var extendedProjects *proj.Projects
var targetFolder string
var projectName string

func main() {
	runInterruptHandler()

	async := flag.Bool("async", false, "Set async mode")
	cat := flag.Bool("cat", false, "Print all the things")
	makeAbout := flag.Bool("about", false, "Create example about yml")
	cloneLists := flag.Bool("cloneLists", true, "Clone lists specified in config")
	flag.Parse()
	state.CloneLists = *cloneLists
	state.Async = *async
	run(*makeAbout, *cat)
}

func run(makeAbout bool, cat bool) {
	configFiles, err := proj.GetConfigs()
	handleError(err)
	if len(configFiles) == 0 {
		handleError(fmt.Errorf("No config files found"))
	}
	for _, configFile := range configFiles {
		configResults, err := proj.RunConfig(configFile)
		handleError(err)
		projects = configResults.Projects
		extendedProjects = configResults.ProjectsExtended
		targetFolder = configResults.TargetFolder
		projectName = configResults.ProjectName

		handleError(util.RunCommand("mkdir", targetFolder))

		cloneChan := make(chan error, len(*projects)+len(*extendedProjects))
		cloneAndMove := func(ps *proj.Projects, prefix string) {
			for k, v := range *ps {
				fmt.Println(prefix + " " + k + "...")
				if state.Async {
					go v.CloneAndMove(cloneChan)
				} else {
					v.CloneAndMove(cloneChan)
				}
			}
		}
		cloneAndMove(projects, "Cloning")
		cloneAndMove(extendedProjects, "Cloning e")

		for i := 0; i < len(*projects)+len(*extendedProjects); i++ {
			handleError(<-cloneChan)
		}

		handleError(proj.RepoCheck(projects, configResults.RepoCheckPathExceptions))
		handleError(proj.RepoCheck(extendedProjects, configResults.RepoCheckPathExceptions))

		fmt.Println()

		ingester := proj.Ingester{projects}
		if err = ingester.IngestAll(true); err != nil {
			fmt.Println("Ingest error:", err.Error())
		}
		ingester = proj.Ingester{extendedProjects}
		if err = ingester.IngestAll(true); err != nil {
			fmt.Println("Ingest e error:", err.Error())
		}

		fmt.Println()
		fmt.Println("Cleaning...", cleanup())
		fmt.Println()

		generatedDepList := projects.GetAllDependencies()
		generatedExtendedDepList := extendedProjects.GetAllDependencies()
		aboutDepList := configResults.AboutDepList
		softwareDepList := configResults.SoftwareDepList

		deps.RemoveExactDuplicates(&generatedDepList, &generatedExtendedDepList, &aboutDepList, &softwareDepList)

		deps.SwapShaVersions(configResults.ShaStore, &generatedDepList, &generatedExtendedDepList, &aboutDepList, &softwareDepList)

		deps.RemoveExceptions(configResults.DepExceptions, &generatedDepList, &generatedExtendedDepList)

		condGeneratedDepList := generatedDepList.Clone()

		deps.CondenseBundles(configResults.Bundles, &condGeneratedDepList)

		var aboutMissing, aboutExtra, aboutGood, softwareMissing, softwareExtra, softwareGood *deps.GenericDependencies
		if state.ComparingAbout {
			aboutMissing, aboutExtra, aboutGood = deps.CompareSimple(&aboutDepList, &condGeneratedDepList)
			aboutMissing.RemoveDuplicatesByProject()
		}

		combined := append(generatedDepList, generatedExtendedDepList...)
		if state.ComparingSoftwareList {
			softwareMissing, softwareExtra, softwareGood = deps.CompareByProject(&softwareDepList, &combined)
		}

		exist, err := util.Exists("report")
		handleError(err)
		if !exist {
			handleError(util.RunCommand("mkdir", "report"))
		}
		if makeAbout {
			aboutList := condGeneratedDepList.Clone()
			aboutList.RemoveDuplicatesByProject()
			writeAboutYML(&aboutList)
		}
		writeIssuesYML(cat)
		writeDependenciesYML(&combined, cat)
		if state.ComparingAbout {
			writeAboutCompare(aboutMissing, aboutExtra, aboutGood)
		}
		if state.ComparingSoftwareList {
			writeSoftwareCompare(softwareMissing, softwareExtra, softwareGood)
		}
	}
}

func writeFile(str string, path string) {
	file, err := os.Create(path)
	handleError(err)
	defer file.Close()
	writer := bufio.NewWriter(file)
	defer writer.Flush()
	writer.WriteString(str)
}

func runInterruptHandler() {
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		cleanup()
		os.Exit(1)
	}()
}

func writeAboutCompare(aboutMissing, aboutExtra, aboutGood *deps.GenericDependencies) {
	fmt.Println("Writing about compare file...")
	var splitBy func(d *deps.GenericDependency) string
	if state.AboutByLanguage {
		splitBy = func(d *deps.GenericDependency) string { return string(d.GetLanguage()) }
	} else if state.AboutSimple {
		splitBy = func(d *deps.GenericDependency) string { return "" }
	}
	combined := append(*aboutExtra, *aboutMissing...)
	writeFile(string(report.GenerateDiffFileDat(report.GenerateDiffMap(aboutMissing, aboutExtra, aboutGood, splitBy), "Extra in about yaml", "Missing in about yaml", "good", combined, true)), generateFileName("report/about-diff.txt"))
}

func writeSoftwareCompare(listMissing, listExtra, listGood *deps.GenericDependencies) {
	fmt.Println("Writing software compare file...")
	splitBy := func(d *deps.GenericDependency) string { return d.GetProject() }
	combined := append(*listExtra, *listMissing...)
	writeFile(string(report.GenerateDiffFileDat(report.GenerateDiffMap(listMissing, listExtra, listGood, splitBy), "Extra in spreadsheet", "Missing in spreadsheet", "good", combined, false)), generateFileName("report/list-diff.txt"))
}

func writeIssuesYML(cat bool) {
	fmt.Println("Writing issues file...")
	ymlDat, err := report.GenerateIssuesYaml(projects)
	handleError(err)
	if cat {
		fmt.Printf("#####################\n%s#####################\n", report.YmlIssuesHeader+string(ymlDat))
	}
	writeFile(report.YmlIssuesHeader+string(ymlDat), generateFileName("report/issues.yml"))
}

func writeDependenciesYML(depens *deps.GenericDependencies, cat bool) {
	fmt.Println("Writing dependecies file...")
	ymlDat, err := report.GenerateDependenciesYaml(depens)
	handleError(err)
	if cat {
		fmt.Printf("#####################\n%s#####################\n", report.YmlDependenciesHeader+string(ymlDat))
	}
	writeFile(report.YmlDependenciesHeader+string(ymlDat), generateFileName("report/dependencies.yml"))
}

func writeAboutYML(depens *deps.GenericDependencies) {
	fmt.Println("Writing about file...")
	ymlDat, err := report.GenerateStacksYaml(depens)
	handleError(err)
	writeFile(report.YmlAboutHeader+string(ymlDat), generateFileName("report/about.yml"))
}

func generateFileName(fileName string) string {
	if strings.TrimSpace(fileName) == "" {
		return fileName
	}
	parts := strings.Split(fileName, "/")
	res := ""
	for i := 0; i < len(parts)-1; i++ {
		res += parts[i] + "/"
	}
	res += projectName + "-"
	return res + parts[len(parts)-1]
}

func cleanup() error {
	return os.RemoveAll(targetFolder)
}

func handleError(err error) {
	if err != nil {
		cleanup()
		log.Fatalln(err)
	}
}
