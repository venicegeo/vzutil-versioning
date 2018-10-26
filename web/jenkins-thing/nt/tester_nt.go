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
package nt

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"

	nt "github.com/venicegeo/pz-gocommon/gocommon"
)

var urlArgs = regexp.MustCompile(`\?[^/]+`)

type HTTP interface {
	HTTP(string, string, [][2]string, io.Reader) (int, []byte, http.Header, error)
}
type RealHTTP struct{}

func (r *RealHTTP) HTTP(requestType string, url string, header [][2]string, toSend io.Reader) (int, []byte, http.Header, error) {
	return nt.HTTP(requestType, url, header, toSend)
}

type TestHTTP_FS struct {
	Testdir string
}

func (t *TestHTTP_FS) HTTP(requestType string, url string, header [][2]string, toSend io.Reader) (int, []byte, http.Header, error) {
	index := strings.Index(url, "://")
	if index != -1 {
		url = url[index+3:]
	}
	if !strings.HasSuffix(url, "/") {
		url += "/"
	}
	url += requestType + ".json"
	if t.Testdir != "" && !strings.HasSuffix(t.Testdir, "/") {
		t.Testdir += "/"
	}
	url = urlArgs.ReplaceAllString(url, "")
	url = t.Testdir + url
	dat, err := ioutil.ReadFile(url)
	if err != nil {
		return 0, nil, nil, err
	}
	var res struct {
		Code   int                    `json:"code"`
		String string                 `json:"string"`
		Json   map[string]interface{} `json:"json"`
	}
	if err := nt.UnmarshalNumber(bytes.NewReader(dat), &res); err != nil {
		return 0, nil, nil, err
	}
	if res.String != "" {
		return res.Code, []byte(res.String), nil, nil
	}
	resDat, err := json.Marshal(res.Json)
	if err != nil {
		return 0, nil, nil, err
	}
	return res.Code, resDat, nil, nil
}

type mapElem struct {
	code int
	body string
}
type TestHTTP_Map struct {
	urls map[string]mapElem
}

func NewMap() *TestHTTP_Map {
	return &TestHTTP_Map{map[string]mapElem{}}
}

func fixurl(url string) string {
	parts := strings.SplitN(url, "://", 2)
	url = parts[len(parts)-1]
	index := strings.LastIndex(url, "?")
	if index > 0 {
		url = url[:index]
	}
	return url
}

func (t *TestHTTP_Map) Add(path string, code int, body string) {
	t.urls[fixurl(path)] = mapElem{code, body}
}

func (t *TestHTTP_Map) HTTP(requestType string, url string, header [][2]string, toSend io.Reader) (int, []byte, http.Header, error) {
	//	fmt.Println("Looking for", fixurl(url))
	//	for k, _ := range t.urls {
	//		fmt.Println(k)
	//	}
	elem, exists := t.urls[fixurl(url)]
	if !exists {
		return 404, nil, nil, nil
	}
	return elem.code, []byte(elem.body), nil, nil
}
