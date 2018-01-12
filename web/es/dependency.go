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

package es

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
)

type Dependency struct {
	Name     string `json:"name"`
	Version  string `json:"version"`
	Language string `json:"language"`
}

func (d *Dependency) GetHashSum() string {
	tmp := md5.Sum([]byte(fmt.Sprintf("%s:%s:%s", d.Name, d.Version, d.Language)))
	return hex.EncodeToString(tmp[:])
}

func (d *Dependency) String() string {
	return fmt.Sprintf("%s:%s:%s", d.Name, d.Version, d.Language)
}
