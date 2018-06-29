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
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"syscall"
	"time"

	com "github.com/venicegeo/vzutil-versioning/common"
	dep "github.com/venicegeo/vzutil-versioning/common/dependency"
	"github.com/venicegeo/vzutil-versioning/common/issue"
	r "github.com/venicegeo/vzutil-versioning/single/resolve"
	"github.com/venicegeo/vzutil-versioning/single/util"
)

type stringarr []string

func (stringarr) String() string {
	return ""
}

func (a *stringarr) Set(value string) error {
	*a = append(*a, value)
	return nil
}

var cleanup func()

func main() {
	var err error

	runInterruptHandler()

	var scan bool
	var all bool
	var includeTest bool
	var files stringarr
	flag.BoolVar(&scan, "scan", false, "Scan for dependency files")
	flag.BoolVar(&all, "all", false, "Run against all found dependency files")
	flag.BoolVar(&includeTest, "testing", false, "Include testing dependencies")
	flag.Var(&files, "f", "Add file to scan")
	flag.Parse()
	info := flag.Args()

	if scan && all {
		fmt.Println("Cannot run in scan and resolve mode")
		os.Exit(1)
	} else if all && len(files) != 0 {
		fmt.Println("Cannot scan all and certain files")
		os.Exit(1)
	} else if len(info) != 2 {
		fmt.Println("The program arguments were incorrect. Usage: single [options] [org/repo] [sha]")
		os.Exit(1)
	}

	name := strings.Split(info[0], "/")[1]
	location, sha, err := cloneAndCheckout(info[0], info[1], name)
	cleanup = func() { util.RunCommand("rm", "-rf", strings.TrimSuffix(location, name)) }
	defer cleanup()
	if err != nil {
		cleanup()
		fmt.Println("Error checking out:", err)
		os.Exit(1)
	}

	if scan {
		files, err := modeScan(location, name, includeTest)
		cleanup()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		str, err := util.GetJson(map[string]interface{}{"files": files})
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Println(str)
	} else {
		if all {
			if files, err = modeScan(location, name, includeTest); err != nil {
				cleanup()
				fmt.Println(err)
				os.Exit(1)
			}
		}
		deps, issues, err := modeResolve(location, name, files, includeTest)
		cleanup()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		sdeps := make([]string, len(deps), len(deps))
		for i, d := range deps {
			sdeps[i] = d.FullString()
		}
		sissues := make([]string, len(issues), len(issues))
		for i, is := range issues {
			sissues[i] = is.String()
		}
		sort.Strings(sdeps)
		sort.Strings(sissues)
		if dat, err := util.GetJson(com.RepositoryDependencies{info[0], sha, "", sdeps, sissues, files}); err != nil {
			fmt.Println(err)
			os.Exit(1)
		} else {
			fmt.Println(string(dat))
		}
	}
}

func modeScan(location, name string, test bool) ([]string, error) {
	fullLocation := fmt.Sprintf("%s/%s", location, name)
	fileLocations := []string{}
	knownFiles := []string{"pom.xml", "glide.yaml", "package.json", "environment.yml", "requirements.txt", "meta.yaml"}
	knownTestFiles := []string{"requirements-dev.txt", "environment-dev.yml"}
	visit := func(path string, f os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if f.IsDir() {
			return nil
		}
		if util.IsVendorPath(path, fullLocation) || util.IsDotGitPath(path, fullLocation) {
			return nil
		}
		found := false
		for _, k := range knownFiles {
			if k == f.Name() {
				fileLocations = append(fileLocations, path)
				found = true
				break
			}
		}
		if test && !found {
			for _, k := range knownTestFiles {
				if k == f.Name() {
					fileLocations = append(fileLocations, path)
					break
				}
			}
		}
		return nil
	}
	if err := filepath.Walk(location, visit); err != nil {
		return nil, err
	}
	for i, f := range fileLocations {
		fileLocations[i] = strings.TrimPrefix(strings.TrimPrefix(f, fullLocation), "/")
	}
	return fileLocations, nil
}

var getFile = regexp.MustCompile(`^\/?(?:[^\/]+\/)*(.+)$`)

var fileToFunc = map[string]func(string, bool) ([]*dep.GenericDependency, []*issue.Issue, error){
	"glide.yaml":           r.ResolveGlideYaml,
	"package.json":         r.ResolvePackageJson,
	"environment.yml":      r.ResolveEnvironmentYml,
	"environment-dev.yml":  r.ResolveEnvironmentYml,
	"requirements.txt":     r.ResolveRequirementsTxt,
	"requirements-dev.txt": r.ResolveRequirementsTxt,
	"meta.yaml":            r.ResolveMetaYaml,
	"pom.xml":              r.ResolvePomXml,
}

func modeResolve(location, name string, files []string, test bool) ([]*dep.GenericDependency, []*issue.Issue, error) {
	var deps []*dep.GenericDependency
	var issues []*issue.Issue
	for _, f := range files {
		matches := getFile.FindStringSubmatch(f)
		if len(matches) != 2 {
			fmt.Printf("File [%f] could not be parsed\n", f)
			cleanup()
			os.Exit(1)
		}
		full := fmt.Sprintf("%s/%s/%s", location, name, f)
		funcc, ok := fileToFunc[matches[1]]
		if !ok {
			fmt.Printf("Could not scan file [%s]\n", f)
			cleanup()
			os.Exit(1)
		}
		d, i, e := funcc(full, test)
		if e != nil {
			return nil, nil, e
		}
		deps = append(deps, d...)
		issues = append(issues, i...)
	}
	dep.RemoveExactDuplicates(&deps)
	return deps, issues, nil
}

func cloneAndCheckout(full_name, checkout, name string) (t string, sha string, err error) {
	t = fmt.Sprintf("%d", time.Now().UnixNano())
	var cmdRet util.CmdRet
	if cmdRet = util.RunCommand("mkdir", t); cmdRet.IsError() {
		return t, "", cmdRet.Error()
	}
	if t, err = filepath.Abs(t); err != nil {
		return t, "", err
	}
	rest := t
	t = fmt.Sprintf("%s/%s", t, name)
	if cmdRet = util.RunCommand("git", "clone", "https://github.com/"+full_name, t); cmdRet.IsError() {
		return t, "", cmdRet.Error()
	}
	if cmdRet = util.RunCommand("git", "-C", t, "checkout", checkout); cmdRet.IsError() {
		return t, "", cmdRet.Error()
	}
	if cmdRet = util.RunCommand("git", "-C", t, "rev-parse", "HEAD"); cmdRet.IsError() {
		return t, "", cmdRet.Error()
	}
	return strings.TrimSuffix(rest, "/"), strings.TrimSpace(cmdRet.Stdout), nil
}

func runInterruptHandler() {
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		os.Exit(1)
	}()
}
