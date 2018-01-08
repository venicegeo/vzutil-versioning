// Copyright 2016, RadiantBlue Technologies, Inc.
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

package piazza

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"unicode"
)

func StructStringToInterface(stru string) (interface{}, error) {
	data := []byte(stru)
	source := (*json.RawMessage)(&data)
	var res interface{}
	err := json.Unmarshal(*source, &res)
	return res, err
}
func StructInterfaceToString(stru interface{}) (string, error) {
	data, err := json.MarshalIndent(stru, " ", "   ")
	return string(data), err
}

func GetVarsFromStruct(struc interface{}) (map[string]interface{}, error) {
	input, ok := struc.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("Structure is not of type map[string]interface{}, currently: %T", struc)
	}
	return getVarsFromStructHelper(input, map[string]interface{}{}, []string{}), nil
}
func getVarsFromStructHelper(inputObj map[string]interface{}, res map[string]interface{}, path []string) map[string]interface{} {
	for k, v := range inputObj {
		wasMap := false
		switch v.(type) {
		case map[string]interface{}:
			wasMap = true
			path = append(path, k)
			res = getVarsFromStructHelper(v.(map[string]interface{}), res, path)
		default:
			temp := ""
			for i := 0; i < len(path); i++ {
				temp += path[i] + "."
			}
			res[fmt.Sprintf("%s%s", temp, k)] = v
		}
		if wasMap {
			path = path[:len(path)-1]
		}
	}
	return res
}

func ValueIsValidArray(value interface{}) bool {
	s := reflect.ValueOf(value)
	return s.Kind() == reflect.Array || s.Kind() == reflect.Slice
}
func CharAt(str string, index int) string {
	return str[index : index+1]
}
func RemoveWhitespace(str string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, str)
}
func InsertString(str, insert string, index int) string {
	return str[:index] + insert + str[index:]
}
func SplitString(str string, whereToSplit int) (string, string) {
	return str[:whereToSplit], str[whereToSplit:]
}
