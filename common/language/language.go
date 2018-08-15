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
package language

import (
	"strings"
)

type Language string

func (l *Language) String() string {
	return string(*l)
}

var LangToFile = map[Language][]string{
	Java:       []string{"pom.xml"},
	JavaScript: []string{"package.json"},
	Go:         []string{"glide.yaml"},
	Python:     []string{"requirements.txt"},
	Conda:      []string{"environment.yml", "meta.yaml"},
}
var FileToLang = map[string]Language{
	"pom.xml":          Java,
	"package.json":     JavaScript,
	"glide.yaml":       Go,
	"requirements.txt": Python,
	"environment.yml":  Conda,
	"meta.yaml":        Conda,
}

const Java, JavaScript, Go, Python, Conda, Unknown Language = "java", "javascript", "go", "python", "conda", "unknown"

func GetLanguage(lang string) Language {
	lang = strings.ToLower(strings.TrimSuffix(lang, "stack"))
	switch lang {
	case string(Java):
		return Java
	case string(JavaScript):
		return JavaScript
	case string(Go):
		return Go
	case string(Python):
		return Python
	case string(Conda):
		return Conda
	default:
		return Unknown
	}
}
