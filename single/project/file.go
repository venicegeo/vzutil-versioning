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
package project

import "io/ioutil"
import "fmt"

var _ IFileReader = (*FileReader)(nil)
var _ IFileReader = (*MockFileReader)(nil)

type IFileReader interface {
	Read(location string) ([]byte, error)
}

type FileReader struct{}

func (f *FileReader) Read(location string) ([]byte, error) {
	return ioutil.ReadFile(location)
}

type MockFileReader struct{}

func (f *MockFileReader) Read(location string) ([]byte, error) {
	switch location {
	case "environment.yml":
		return []byte(pythontest1), nil
	default:
		return []byte{}, fmt.Errorf("Unknown file")
	}
}

const pythontest1 string = `
name: python-test1-name
dependencies:
  - solid=1.0.0
  - weak>=1.10.0
  - wildcard=1.2.*
  - missing
  - pip: 
    - pipsolid=2.0.0
    - pipweak>=2.10.0
    - pipwildcard=2.2.*
    - pipmissing
`
