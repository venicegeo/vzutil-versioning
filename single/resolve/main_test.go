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
package resolve

import (
	"fmt"
	"os"
	"reflect"
	"testing"

	d "github.com/venicegeo/vzutil-versioning/common/dependency"
	i "github.com/venicegeo/vzutil-versioning/common/issue"
)

var resolver *Resolver

var testData = map[string]string{}
var testResults = map[string]ResolveResult{}
var testFunctions = map[string]func(string, bool) (d.Dependencies, i.Issues, error){}
var testCount = map[string]int{}

func read(file string) ([]byte, error) {
	return []byte(testData[file]), nil
}

func TestMain(m *testing.M) {
	resolver = NewResolver(read)
	os.Exit(m.Run())
}

func run(name string, t *testing.T) {
	count := testCount[name]
	for i := 0; i < count; i++ {
		filename := fmt.Sprintf("%s-%d", name, i+1)
		fmt.Println("Testing", filename)
		expected := testResults[filename]
		d, i, e := testFunctions[name](filename, true)
		if !reflect.DeepEqual(e, expected.err) {
			t.Fatal(e, "not equal to", expected.err)
		} else if !reflect.DeepEqual(d, expected.deps) {
			t.Fatal(d, "not equal to", expected.deps)
		} else if !reflect.DeepEqual(i, expected.issues) {
			t.Fatal(i, "not equal to", expected.issues)
		}
	}
}

func addTest(name string, data string, result ResolveResult, function func(string, bool) (d.Dependencies, i.Issues, error)) {
	if _, ok := testCount[name]; !ok {
		testCount[name] = 1
	} else {
		testCount[name]++
	}
	filename := fmt.Sprintf("%s-%d", name, testCount[name])
	testData[filename] = data
	testResults[filename] = result
	testFunctions[name] = function
}
