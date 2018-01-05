/*
Copyright 2016, RadiantBlue Technologies, Inc.

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

package piazza

import (
	"encoding/base64"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
)

type HeaderBuilder struct {
	header [][2]string
}

const GET, PUT, POST, DELETE, HEAD = "GET", "PUT", "POST", "DELETE", "HEAD"

func NewHeaderBuilder() *HeaderBuilder {
	return &HeaderBuilder{[][2]string{}}
}
func (builder *HeaderBuilder) AddJsonContentType() *HeaderBuilder {
	builder.AddHeader("Content-Type", "application/json")
	return builder
}
func (builder *HeaderBuilder) AddBasicAuth(username, password string) *HeaderBuilder {
	auth := GetBasicAuthHeader(username, password)
	builder.AddHeader(auth[0], auth[1])
	return builder
}
func (builder *HeaderBuilder) AddHeader(key, value string) *HeaderBuilder {
	builder.header = append(builder.header, [2]string{key, value})
	return builder
}
func (builder *HeaderBuilder) GetHeader() [][2]string {
	return builder.header
}
func HTTP(requestType, url string, headers [][2]string, toSend io.Reader) (int, []byte, http.Header, error) {
	if !hasProtocol.MatchString(url) {
		url = "https://" + url
	}
	req, err := http.NewRequest(requestType, url, toSend)
	if err != nil {
		return 0, nil, nil, err
	}
	for _, v := range headers {
		req.Header.Set(v[0], v[1])
	}
	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		return 0, nil, nil, err
	}

	defer func() {
		err = response.Body.Close()
		if err != nil {
			// TODO: no way to handle error in a defer!
			panic(err)
		}
	}()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return response.StatusCode, nil, nil, err
	}
	return response.StatusCode, body, response.Header, nil
}

/*
func HTTPReq(requestType, url string, headers [][2]string, toSend io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(requestType, url, toSend)
	if err != nil {
		return nil, err
	}
	for _, v := range headers {
		req.Header.Set(v[0], v[1])
	}
	client := &http.Client{}
	return client.Do(req)
}
*/

func GetBasicAuthHeader(username, password string) [2]string {
	username = strings.TrimSpace(username)
	password = strings.TrimSpace(password)
	res := username + ":" + password
	enc := base64.URLEncoding.EncodeToString([]byte(res))
	return [2]string{"Authorization", "Basic " + enc}
}
func GetValueFromHeader(header http.Header, field string) string {
	return header.Get(field)
}
