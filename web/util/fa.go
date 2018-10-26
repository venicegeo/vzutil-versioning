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

type FA struct {
	currentState string
	trans        map[string][]fatrans
	every        []fatrans
}

type fatrans struct {
	node  func(string) bool
	trans string
}

func NewFA() *FA {
	return &FA{"", map[string][]fatrans{}, []fatrans{}}
}
func (fa *FA) Start(name string) *FA {
	fa.currentState = name
	return fa
}
func (fa *FA) Add(name string, node func(string) bool, trans string) {
	if _, ok := fa.trans[name]; !ok {
		fa.trans[name] = []fatrans{}
	}
	fa.trans[name] = append(fa.trans[name], fatrans{node, trans})
}
func (fa *FA) Every(node func(string) bool, trans string) {
	fa.every = append(fa.every, fatrans{node, trans})
}
func (fa *FA) Next(line string) {
	for _, trans := range append(fa.trans[fa.currentState], fa.every...) {
		if trans.node(line) {
			fa.currentState = trans.trans
			break
		}
	}
}
