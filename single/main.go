/*
Copyright 2019, RadiantBlue Technologies, Inc.

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
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"syscall"
	"time"

	com "github.com/venicegeo/vzutil-versioning/common"
	d "github.com/venicegeo/vzutil-versioning/common/dependency"
	h "github.com/venicegeo/vzutil-versioning/common/history"
	i "github.com/venicegeo/vzutil-versioning/common/issue"
	r "github.com/venicegeo/vzutil-versioning/single/resolve"
	"github.com/venicegeo/vzutil-versioning/single/util"
)

type stringarr []string

var scan bool
var all bool
var includeTest bool
var files stringarr
var full_name string
var name string
var localMode bool
var history bool

var cleanup func()

var resolver *r.Resolver

func main() {
	var err error
	timestamp := time.Now()

	runInterruptHandler()

	flag.BoolVar(&history, "history", false, "Generate history tree")
	flag.BoolVar(&localMode, "local", false, "Run in local mode")
	flag.BoolVar(&scan, "scan", false, "Scan for dependency files")
	flag.BoolVar(&all, "all", false, "Run against all found dependency files")
	flag.BoolVar(&includeTest, "testing", true, "Include testing dependencies")
	flag.Var(&files, "f", "Add file to scan")
	flag.Parse()
	info := flag.Args()

	if (scan && all) || (scan && history) || (all && history) {
		fmt.Println("Can only run in one mode at a time")
		os.Exit(1)
	} else if all && len(files) != 0 {
		fmt.Println("Cannot scan all and certain files")
		os.Exit(1)
	} else if len(files) == 0 && !(scan || all || history) {
		fmt.Println("Must give a run paramater")
		os.Exit(1)
	} else if (localMode || history) && len(info) != 1 || !(localMode || history) && len(info) != 2 {
		fmt.Println("The program arguments were incorrect. Usage: single [options] [org/repo] [sha]")
		os.Exit(1)
	}

	resolver = r.NewResolver(ioutil.ReadFile)
	genFileToFunc()

	var location, sha string
	var refs []string

	if len(info) == 1 {
		info = append(info, "master") //TODO
	}

	full_name = info[0]
	if !localMode {
		name = strings.SplitN(info[0], "/", 2)[1]
		location, sha, refs, err = cloneAndCheckout(info[0], info[1], name)
	} else {
		name = ""
		location = full_name
		sha = "Local"
		refs = []string{}
	}
	cleanup = func() {
		if !localMode {
			util.RunCommand("rm", "-rf", strings.TrimSuffix(location, name))
		}
	}
	defer cleanup()
	if err != nil {
		cleanup()
		fmt.Println("Error checking out:", err)
		os.Exit(1)
	}

	if history {
		tree, err := modeHistory(location, name)
		cleanup()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		str, err := util.GetJson(tree)
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}
		fmt.Println(str)
	} else if scan {
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
		if dat, err := util.GetJson(com.DependencyScan{full_name, name, sha, refs, deps, issues.SSlice(), files, timestamp}); err != nil {
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

var fileToFunc map[string]func(string, bool) (d.Dependencies, i.Issues, error)

func genFileToFunc() {
	fileToFunc = map[string]func(string, bool) (d.Dependencies, i.Issues, error){
		"glide.yaml":           resolver.ResolveGlideYaml,
		"package.json":         resolver.ResolvePackageJson,
		"environment.yml":      resolver.ResolveEnvironmentYml,
		"environment-dev.yml":  resolver.ResolveEnvironmentYml,
		"requirements.txt":     resolver.ResolveRequirementsTxt,
		"requirements-dev.txt": resolver.ResolveRequirementsTxt,
		"meta.yaml":            resolver.ResolveMetaYaml,
		"pom.xml":              resolver.ResolvePomXml,
	}
}

func modeResolve(location, name string, files []string, test bool) (d.Dependencies, i.Issues, error) {
	var deps d.Dependencies
	var issues i.Issues
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
			return nil, nil, fmt.Errorf("%s: %s", f, e)
		}
		deps = append(deps, d...)
		issues = append(issues, i...)
	}
	d.RemoveExactDuplicates(&deps)
	sort.Sort(deps)
	return deps, issues, nil
}

func cloneAndCheckout(full_name, checkout, name string) (string, string, []string, error) {
	t := fmt.Sprintf("%d", time.Now().UnixNano())
	var err error
	var cmdRet util.CmdRet
	if cmdRet = util.RunCommand("mkdir", t); cmdRet.IsError() {
		return t, "", nil, cmdRet.Error()
	}
	if t, err = filepath.Abs(t); err != nil {
		return t, "", nil, err
	}
	rest := t
	t = fmt.Sprintf("%s/%s", t, name)
	if cmdRet = util.RunCommand("git", "clone", "https://github.com/"+full_name, t); cmdRet.IsError() {
		return t, "", nil, cmdRet.Error()
	}

	util.RunCommand("bash", "-c", fmt.Sprintf(`git -C %s branch -r | grep -v '\->' | while read remote; do git -C %s branch --track "${remote#origin/}" "$remote"; done`, t, t))
	util.RunCommand("git", "-C", t, "fetch", "--all")
	util.RunCommand("git", "-C", t, "pull", "--all")

	if cmdRet = util.RunCommand("git", "-C", t, "checkout", checkout); cmdRet.IsError() {
		return t, "", nil, cmdRet.Error()
	}
	if cmdRet = util.RunCommand("git", "-C", t, "rev-parse", "HEAD"); cmdRet.IsError() {
		return t, "", nil, cmdRet.Error()
	}
	sha := strings.TrimSpace(cmdRet.Stdout)
	if cmdRet = util.RunCommand("git", "-C", t, "show-ref", "-d"); cmdRet.IsError() {
		return t, "", nil, cmdRet.Error()
	}
	tmp := map[string]string{}
	lines := strings.Split(cmdRet.Stdout, "\n")
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}
		parts := strings.Split(strings.TrimSpace(l), " ")
		sha := strings.TrimSuffix(parts[1], `^{}`)
		if !strings.HasSuffix(sha, "/HEAD") {
			tmp[strings.Replace(sha, "remotes/origin", "heads", -1)] = parts[0]
		}
	}
	refs := []string{}
	for k, v := range tmp {
		if v == sha {
			refs = append(refs, k)
		}
	}

	return strings.TrimSuffix(rest, "/"), sha, refs, nil
}

func runInterruptHandler() {
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		os.Exit(1)
	}()
}

func (stringarr) String() string {
	return ""
}
func (a *stringarr) Set(value string) error {
	*a = append(*a, value)
	return nil
}

func modeHistory(location, name string) (h.HistoryTree, error) {
	gitLocation := location + "/" + name
	shaToBranch := map[string]string{}
	dat, err := exec.Command("git", "-C", gitLocation, "ls-remote", "--heads", "origin").Output()
	if err != nil {
		return nil, err
	}
	{
		var parts []string
		for _, line := range strings.Split(strings.TrimSpace(string(dat)), "\n") {
			parts = strings.Split(line, "\t")
			shaToBranch[parts[0]] = strings.TrimPrefix(parts[1], "refs/heads/")
		}
	}
	//fmt.Println("Branches found:", len(shaToBranch))
	dat, err = exec.Command("git", "-C", gitLocation, "log", "--all", `--pretty=format:%H|%P`).Output()
	if err != nil {
		return nil, err
	}
	tree := h.HistoryTree{}
	lines := strings.Split(strings.TrimSpace(string(dat)), "\n")
	parts := make([][]string, len(lines))
	for i := 0; i < len(lines); i++ {
		parts[i] = strings.Split(lines[i], "|")
		tree[parts[i][0]] = &h.HistoryNode{parts[i][0], []string{}, []string{}, []string{}, 0, false, false}
	}
	for i := len(parts) - 1; i >= 0; i-- {
		line := parts[i]
		node := tree[line[0]]
		parents := strings.Split(line[1], " ")
		for _, p := range parents {
			if p == "" {
				continue
			}
			node.Parents = append(node.Parents, p)
			tree[p].Children = append(tree[p].Children, node.Sha)
		}
	}
	dat, err = exec.Command("bash", "-c", fmt.Sprintf(`git -C "%s" show HEAD --pretty=format:"%s" | head -1`, gitLocation, "%H")).Output()
	if err != nil {
		return nil, err
	}
	HEADsha := strings.TrimSpace(string(dat))
	tree[HEADsha].IsHEAD = true
	if err := fillTreeWithName(tree, gitLocation, HEADsha, shaToBranch[HEADsha]); err != nil {
		return nil, err
	}
	//	fmt.Println(HEADsha, shaToBranch[HEADsha])
	//	fmt.Println(tree.GetRoots())
	for sha, branch := range shaToBranch {
		if err := fillTreeWithName(tree, gitLocation, sha, branch); err != nil {
			return nil, err
		}
	}
	unknownBranchCount := 0
	for _, node := range tree {
		if len(node.Names) == 0 && len(node.Children) == 1 && len(tree[node.Children[0]].Parents) == 2 {
			unknownBranchCount++
			newBranchName := fmt.Sprintf("!DELETED - %d", unknownBranchCount)
			shaToBranch[node.Sha] = newBranchName
			if err := fillTreeWithName(tree, gitLocation, node.Sha, newBranchName); err != nil {
				return nil, err
			}
		}
	}

	//	fmt.Println(shaToBranch)

	//	for sha, name := range shaToBranch {
	//	tree[sha].Names = append(tree[sha].Names, name)
	//		fmt.Println("Starting", sha, name)
	//		testGiveName(tree, sha, name)
	//	}

	//fmt.Println("With deleted:", len(shaToBranch))
	//for true {
	//	foundSomething := false
	//	for _, node := range tree {
	//		if len(node.Children) > 1 && len(node.Names) == len(node.Children) {
	//			//fmt.Println("Node", node.Sha[:7], "has names", node.Names)
	//			if err := filterOutName(tree, gitLocation, node.Sha, shaToBranch); err != nil {
	//				return nil, err
	//			}
	//			foundSomething = true
	//				break
	//		}
	//	}
	//		if !foundSomething {
	//			break
	//		}
	//	}
	for sha, _ := range shaToBranch {
		tree[sha].IsStartOfBranch = true
	}
	return tree, nil
}

//func getMostParentParent

func testFillName3(t h.HistoryTree, sha, name string) {

}

func fillTreeWithName(t h.HistoryTree, gitLocation, sha, name string) error {
	dat, err := exec.Command("git", "-C", gitLocation, "log", "--first-parent", `--pretty=format:%H`, sha).Output()
	if err != nil {
		return err
	}
	for _, s := range strings.Split(strings.TrimSpace(string(dat)), "\n") {
		if len(t[s].Names) > 0 {
			continue
		}
		t[s].Names = append(t[s].Names, name)
	}
	return nil
}

func filterOutName(t h.HistoryTree, gitLocation, sha string, shaToBranch map[string]string) error {
	node := t[sha]
	//fmt.Println("Children:", node.Children[0][:7], node.Children[1][:7])
	var branchAName = t[node.Children[0]].Names[0]
	var branchBName = t[node.Children[1]].Names[0]
	var branchASha, branchBSha string
	for s, b := range shaToBranch {
		if b == branchAName {
			branchASha = s
		} else if b == branchBName {
			branchBSha = s
		}
	}
	//fmt.Println(branchASha[:7], branchAName, t[node.Children[0]].Names)
	//fmt.Println(branchBSha[:7], branchBName, t[node.Children[1]].Names)
	var toPurge string
	if t[branchASha].IsHEAD {
		toPurge = branchBName
	} else if t[branchBSha].IsHEAD {
		toPurge = branchAName
	} else {
		dat, err := exec.Command("bash", "-c", fmt.Sprintf(`git -C "%s" log "%s...%s" --oneline --pretty=format:"%s" | tail -1`, gitLocation, branchASha, branchBSha, "%H")).Output()
		if err != nil {
			return err
		}
		realParentSha := strings.TrimSpace(string(dat))
		realParentName := t[realParentSha].Names[0]
		//fmt.Println("Real parent found:", realParentSha[:7], realParentName, t[realParentSha].Names)
		switch realParentName {
		case branchAName:
			toPurge = branchBName
		case branchBName:
			toPurge = branchAName
		default:
			return nil
			return fmt.Errorf("My algo failed :(")
		}
	}
	purgeName(t, sha, toPurge)
	return nil
}

func purgeName(t h.HistoryTree, sha, name string) {
	remove := func(node *h.HistoryNode, s string) bool {
		i := -1
		for ii, name := range node.Names {
			if name == s {
				i = ii
				break
			}
		}
		if i == -1 {
			return false
		}
		node.Names = append(node.Names[:i], node.Names[i+1:]...)
		return true
	}
	if remove(t[sha], name) {
		for _, s := range t[sha].Parents {
			purgeName(t, s, name)
		}
	}
}

// git log 2b201c3...8b2d73a --oneline | tail -1
//0224198 Developing jenkins support
