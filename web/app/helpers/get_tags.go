// Copyright 2018, RadiantBlue Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package helpers

import (
	"errors"
	"os/exec"
	"strings"
	"time"

	u "github.com/venicegeo/vzutil-versioning/web/util"
)

type tagsRunner struct {
	name     string
	fullName string
}

func NewTagsRunner(name, fullName string) *tagsRunner {
	return &tagsRunner{name, fullName}
}

func (tr *tagsRunner) Run() (res map[string]string, err error) {
	res = map[string]string{}
	tmp := map[string]string{}
	tempFolder := u.Format("%d", time.Now().Unix())
	defer func() { exec.Command("rm", "-rf", tempFolder).Run() }()
	targetFolder := u.Format("%s/%s", tempFolder, tr.name)
	if err = exec.Command("mkdir", tempFolder).Run(); err != nil {
		return res, err
	}
	if err = exec.Command("git", "clone", "https://github.com/"+tr.fullName, targetFolder).Run(); err != nil {
		return res, err
	}
	var dat []byte
	if dat, err = exec.Command("git", "-C", targetFolder, "show-ref", "--tags", "-d").Output(); err != nil {
		return res, err
	}
	lines := strings.Split(string(dat), "\n")
	for _, l := range lines {
		if l == "" {
			continue
		}
		shaRef := strings.Split(l, " ")
		if len(shaRef) != 2 {
			return res, errors.New("Problem parsing this line [%" + l + "]")
		}
		tmp[strings.TrimSuffix(shaRef[1], "^{}")] = shaRef[0]
	}
	for k, v := range tmp {
		res[v] = k
	}
	return res, nil

}
