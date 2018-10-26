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

package util

import (
	"crypto/md5"
	"encoding/hex"
	"log"
	"os"
	"strings"
)

func addFSifMissing(url string) string {
	if !strings.HasSuffix(url, "/") {
		url += "/"
	}
	return url
}

func getRequiredEnv(env string) string {
	temp := os.Getenv(env)
	if temp == "" {
		log.Fatal("Missing env var", env)
	}
	return temp
}

func Hash(a string) string {
	tmp := md5.Sum([]byte(a))
	return hex.EncodeToString(tmp[:])
}

func SplitAtAnyTrim(str string, split ...string) []string {
	spl := []string{str}
	for _, s := range split {
		temp := []string{}
		for _, sp := range spl {
			temp = append(temp, strings.Split(sp, s)...)
		}
		spl = temp
	}
	return StringSliceTrimSpaceRemoveEmpty(spl)
}
func StringSliceTrimSpaceRemoveEmpty(s []string) []string {
	u := make([]string, 0, len(s))
	for _, t := range s {
		v := strings.TrimSpace(t)
		if v != "" {
			u = append(u, v)
		}
	}
	return u
}
