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
package util

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

var testPathRe = `^%s\/%s(?:(?:\/)|$){1}.*$`

func IsVendorPath(path string, folderLocation string) bool {
	re := regexp.MustCompile(fmt.Sprintf(testPathRe, folderLocation, `vendor`))
	return re.MatchString(path)
}
func IsDotGitPath(path string, folderLocation string) bool {
	re := regexp.MustCompile(fmt.Sprintf(testPathRe, folderLocation, `\.git`))
	return re.MatchString(path)
}

func Exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}

func RunCommand(name string, arg ...string) ([]byte, error) {
	return exec.Command(name, arg...).Output()
}

func GetJson(i interface{}) (string, error) {
	temp, err := json.MarshalIndent(i, " ", "   ")
	return string(temp), err
}

func StringSliceToLower(s []string) {
	for i, v := range s {
		s[i] = strings.ToLower(v)
	}
}
