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
	"os/exec"
	"strings"
	"time"

	nt "github.com/venicegeo/pz-gocommon/gocommon"
	u "github.com/venicegeo/vzutil-versioning/web/util"
)

func GetBranchSha(name, fullName, branch string) (string, error) {
	code, _, _, err := nt.HTTP(nt.HEAD, u.Format("https://github.com/%s/tree/%s", fullName, branch), nt.NewHeaderBuilder().GetHeader(), nil)
	if err != nil {
		return "", err
	}
	if code != 200 {
		return "", u.Error("Could not verify branch [%s] on repo [%s]", branch, fullName)
	}

	tempFolder := u.Format("%d", time.Now().Unix())
	defer func() { exec.Command("rm", "-rf", tempFolder).Run() }()
	targetFolder := u.Format("%s/%s", tempFolder, name)
	if err = exec.Command("mkdir", tempFolder).Run(); err != nil {
		return "", err
	}
	if err = exec.Command("git", "clone", "https://github.com/"+fullName, targetFolder).Run(); err != nil {
		return "", err
	}
	dat, err := exec.Command("git", "-C", targetFolder, "rev-parse", u.Format("origin/%s", branch)).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(dat)), nil
}
