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
package report

import (
	"fmt"
	"sort"
	"strings"

	deps "github.com/venicegeo/vzutil-versioning/project/dependency"
)

const diff_missing, diff_extra, diff_good = "diff_missing", "diff_extra", "diff_good"

//             proj/stack  miss/ext  deps
type diffMap map[string]map[string][]string

func GenerateDiffMap(missing, extra, good *deps.GenericDependencies, getPS func(*deps.GenericDependency) string) *diffMap {
	diff := diffMap{}
	appendIfNotContains := func(s []string, str string) []string {
		app := true
		for _, v := range s {
			if v == str {
				app = false
				break
			}
		}
		if app {
			s = append(s, str)
		}
		return s
	}
	gen := func(deps *deps.GenericDependencies, key string) {
		for _, dep := range *deps {
			temp := getPS(dep)
			if _, ok := diff[temp]; !ok {
				diff[temp] = map[string][]string{}
			}
			if _, ok := diff[temp][key]; !ok {
				diff[temp][key] = []string{}
			}
			diff[temp][key] = appendIfNotContains(diff[temp][key], dep.String())
		}
	}
	gen(missing, diff_missing)
	gen(extra, diff_extra)
	gen(good, diff_good)
	return &diff
}

func GenerateDiffFileDat(diff *diffMap, extraString, missingString, goodString string, allDeps deps.GenericDependencies, includeProjectName bool) []byte {

	max := func(i ...int) int {
		temp := i[0]
		for _, ii := range i {
			if ii > temp {
				temp = ii
			}
		}
		return temp
	}

	findProjectName := func(dep string) string {
		res := ""
		if strings.TrimSpace(dep) == "" {
			return res
		}
		comps := []string{}
		if allDeps != nil {
			for _, d := range allDeps {
				if strings.EqualFold(strings.TrimSpace(dep), d.String()) {
					comps = append(comps, d.GetProject())
				}
			}
		}
		if len(comps) > 0 {
			res = comps[0]
			if len(comps) > 1 {
				res += fmt.Sprintf(" (%d)", len(comps))
			}
		}
		res = strings.TrimSuffix(res, " | ")
		if res == "" {
			return "Unknown"
		}
		return res
	}

	{ //Size them up
		for _, group := range *diff {
			if _, ok := group[diff_missing]; !ok {
				group[diff_missing] = []string{}
			}
			if _, ok := group[diff_extra]; !ok {
				group[diff_extra] = []string{}
			}
			if _, ok := group[diff_good]; !ok {
				group[diff_good] = []string{}
			}
			length := max(len(group[diff_missing]), max(len(group[diff_extra]), len(group[diff_good])))
			for len(group[diff_missing]) < length {
				group[diff_missing] = append(group[diff_missing], "")
			}
			for len(group[diff_extra]) < length {
				group[diff_extra] = append(group[diff_extra], "")
			}
			sort.Strings(group[diff_good])
			for len(group[diff_good]) < length {
				group[diff_good] = append(group[diff_good], "")
			}
		}
	}

	for groupName, group := range *diff {
		extra := group[diff_extra]
		missing := group[diff_missing]
		(*diff)[groupName][diff_extra], (*diff)[groupName][diff_missing] = similaritySort(extra, missing)
	}

	spaceString := func(str string, size int) string {
		spacesToAdd := max(0, size-len(str))
		leftSpace := spacesToAdd / 2
		rightSpace := spacesToAdd - leftSpace
		for i := 0; i < leftSpace; i++ {
			str = " " + str
		}
		for i := 0; i < rightSpace; i++ {
			str = str + " "
		}
		return str
	}

	columnWidth := 0
	for groupName, group := range *diff {
		columnWidth = max(columnWidth, len(groupName), len(missingString), len(extraString), len(goodString))
		for _, items := range group {
			for _, item := range items {
				columnWidth = max(columnWidth, len(item))
			}
		}
	}
	for _, group := range *diff {
		for _, items := range group {
			for iitem, _ := range items {
				items[iitem] = spaceString(items[iitem], columnWidth)
				items[iitem] = " " + items[iitem] + " "
			}
		}
	}
	columnWidth += 2
	totalWidth := columnWidth*3 + 4

	final := ""

	add := func(str string) {
		final += str
	}
	addlnf := func(str string) {
		add(fmt.Sprintf("|%s|\r\n", str))
	}
	addln := func(str string) {
		add(fmt.Sprintf("%s\r\n", str))
	}
	addf := func(str string) {
		add(fmt.Sprintf("|%s|", str))
	}

	thickSep, thinSep := "", ""
	for i := 0; i < totalWidth; i++ {
		thickSep += "="
		thinSep += "-"
	}

	missingString = spaceString(missingString, columnWidth)
	extraString = spaceString(extraString, columnWidth)
	goodString = spaceString(goodString, columnWidth)
	addlnf(thickSep)

	for groupName, group := range *diff {
		groupName = spaceString(groupName, totalWidth)
		addlnf(groupName)
		addlnf(thinSep)
		addlnf(fmt.Sprintf("%s||%s||%s", goodString, extraString, missingString))
		addlnf(thinSep)
		for i := 0; i < len(group[diff_missing]); i++ {
			addf(fmt.Sprintf("%s||%s||%s", group[diff_good][i], group[diff_extra][i], group[diff_missing][i]))
			if includeProjectName {
				addln(findProjectName(group[diff_missing][i]))
			} else {
				addln("")
			}

		}
		addlnf(thickSep)

	}

	return []byte(final)
}
