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
package util

import (
	"reflect"
	"testing"
)

func TestStringSliceToLower(t *testing.T) {
	actual := []string{"HELLO", "thEre", "user"}
	StringSliceToLower(actual)
	expected := []string{"hello", "there", "user"}
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("Expected: %#v Actual: %#v", expected, actual)
	}
}

func TestStringSliceTrimSpaceRemoveEmpty(t *testing.T) {
	actual := StringSliceTrimSpaceRemoveEmpty([]string{"  hello ", "", "there ", "", "", " use r"})
	expected := []string{"hello", "there", "use r"}
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("Expected: %#v Actual: %#v", expected, actual)
	}
}

func TestSplitAtAnyTrim(t *testing.T) {
	tests := []string{"something correct", "something=wrong", "something==wrong_again", "something =why", "something ==who_did_this"}
	results := [][]string{[]string{"something", "correct"}, []string{"something", "wrong"}, []string{"something", "wrong_again"}, []string{"something", "why"}, []string{"something", "who_did_this"}}
	for i, test := range tests {
		expected := results[i]
		actual := SplitAtAnyTrim(test, " ", "=")
		if !reflect.DeepEqual(actual, expected) {
			t.Errorf("Expected: %#v ACtual: %#v", expected, actual)
		}
	}
}
