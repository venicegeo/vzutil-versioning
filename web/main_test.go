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

package main

import (
	"bytes"
	"fmt"
	"io"
	"testing"
	"time"

	nt "github.com/venicegeo/pz-gocommon/gocommon"
)

func Test1(t *testing.T) {
	go main()
	defer func() { killChan <- true }()
	time.Sleep(time.Second)

	fmt.Println(defaultCall(nt.GET, "http://localhost:20012", nil, t))
	defaultCall(nt.POST, "http://localhost:20012/webhook", bytes.NewReader([]byte(test1)), t)
	defaultCall(nt.POST, "http://localhost:20012/webhook", bytes.NewReader([]byte(test2)), t)
	defaultCall(nt.POST, "http://localhost:20012/webhook", bytes.NewReader([]byte(test3)), t)
	defaultCall(nt.POST, "http://localhost:20012/webhook", bytes.NewReader([]byte(test4)), t)
	time.Sleep(time.Second * 10)
}

func defaultCall(request, url string, toSend io.Reader, t *testing.T) string {
	code, dat, _, err := nt.HTTP(request, url, nt.NewHeaderBuilder().GetHeader(), toSend)
	checkError(err, t)
	status200(code, t)
	return string(dat)
}

func checkError(err error, t *testing.T) {
	if err != nil {
		t.Fatal(err)
	}
}

func status200(code int, t *testing.T) {
	if code != 200 {
		t.Fatal("Code is not 200 but", code)
	}
}

var test1 = `
{
    "ref":"refs/head/master",
    "after":"da09290a7992188358bde11ba6ab7f000b49adaa",
    "repository":{
        "id":1234,
        "name":"pz-gateway",
        "full_name":"venicegeo/pz-gateway",
        "html_url":"https://github.com/venicegeo/pz-gateway",
        "url":"https://github.com/venicegeo/pz-gateway"
    }
}`
var test2 = `
{
    "ref":"refs/head/master",
    "after":"fe1df9f9a7b3b62b7df12c79b23c2a9dff4959cb",
    "repository":{
        "id":1234,
        "name":"pz-gateway",
        "full_name":"venicegeo/pz-gateway",
        "html_url":"https://github.com/venicegeo/pz-gateway",
        "url":"https://github.com/venicegeo/pz-gateway"
    }
}`
var test3 = `
{
    "ref":"refs/head/master",
    "after":"592e467676c0ba2d1e10ad8cf019a0daa047974c",
    "repository":{
        "id":1234,
        "name":"pz-gateway",
        "full_name":"venicegeo/pz-gateway",
        "html_url":"https://github.com/venicegeo/pz-gateway",
        "url":"https://github.com/venicegeo/pz-gateway"
    }
}`
var test4 = `
{
    "ref":"refs/head/master",
    "after":"6d73a8563872dc58a45bb09951d64b43e282ff8d",
    "repository":{
        "id":1234,
        "name":"pz-gateway",
        "full_name":"venicegeo/pz-gateway",
        "html_url":"https://github.com/venicegeo/pz-gateway",
        "url":"https://github.com/venicegeo/pz-gateway"
    }
}`
