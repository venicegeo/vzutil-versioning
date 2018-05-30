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
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/http"
)

//----------------------------------------------------------

type FlightCheck interface {
	Preflight(verb string, url string, obj string) error
	Postflight(code int, obj string) error
}

type Http struct {
	FlightCheck FlightCheck
	ApiKey      string
	BaseUrl     string
	User        string
	Pass        string
}

//----------------------------------------------------------

// note we decode the result even if not a 2xx status
func (h *Http) convertResponseBodyToObject(resp *http.Response, output interface{}) error {
	if output == nil {
		return nil
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	raw, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	err = json.Unmarshal(raw, output)
	return err
}

func (h *Http) convertObjectToReader(input interface{}) (io.Reader, error) {
	if input == nil {
		return nil, nil
	}

	byts, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}

	reader := bytes.NewReader(byts)
	return reader, nil
}

func (h *Http) toJsonString(obj interface{}) string {
	if obj == nil {
		return "{}"
	}

	byts, err := json.Marshal(obj)
	if err != nil {
		return "internal error: unable to marshall into json"
	}
	return string(byts)
}

func (h *Http) doPreflight(verb string, url string, obj interface{}) error {
	if h.FlightCheck == nil {
		return nil
	}
	jsn := h.toJsonString(obj)
	return h.FlightCheck.Preflight(verb, url, jsn)
}

func (h *Http) doPostflight(statusCode int, obj interface{}) error {
	if h.FlightCheck == nil {
		return nil
	}
	jsn := h.toJsonString(obj)
	return h.FlightCheck.Postflight(statusCode, jsn)
}

func (h *Http) doRequest(verb string, url string, reader io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(verb, url, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", ContentTypeJSON)
	if h.User != "" || h.Pass != "" {
		req.SetBasicAuth(h.User, h.Pass)
	} else if h.ApiKey != "" {
		req.SetBasicAuth(h.ApiKey, "")
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (h *Http) doVerb(verb string, endpoint string, input interface{}, output interface{}) (int, error) {
	url := h.BaseUrl + endpoint

	reader, err := h.convertObjectToReader(input)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	err = h.doPreflight(verb, url, input)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	resp, err := h.doRequest(verb, url, reader)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	defer func() {
		err = resp.Body.Close()
		if err != nil {
			// TODO: no way to handle error in a defer!
			panic(err)
		}
	}()

	err = h.convertResponseBodyToObject(resp, output)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	err = h.doPostflight(resp.StatusCode, output)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	return resp.StatusCode, nil
}

//----------------------------------------------------------

// Use these when doing HTTP requests where the inputs and outputs
// are supposed to be JSON strings (for which the caller supplies
// a Go object).

// expects endpoint to return JSON
func (h *Http) Get(endpoint string, output interface{}) (int, error) {
	return h.doVerb("GET", endpoint, nil, output)
}

func (h *Http) Get2(endpoint string, input interface{}, output interface{}) (int, error) {
	return h.doVerb("GET", endpoint, input, output)
}

// expects endpoint to take in and return JSON
func (h *Http) Post(endpoint string, input interface{}, output interface{}) (int, error) {
	return h.doVerb("POST", endpoint, input, output)
}

// expects endpoint to take in and return JSON
func (h *Http) Put(endpoint string, input interface{}, output interface{}) (int, error) {
	return h.doVerb("PUT", endpoint, input, output)
}

// expects endpoint to return nothing
func (h *Http) Delete(endpoint string, output interface{}) (int, error) {
	return h.doVerb("DELETE", endpoint, nil, output)
}

// expects endpoint to take in and return JSON
func (h *Http) Verb(verb string, endpoint string, input interface{}, output interface{}) (int, error) {
	return h.doVerb(verb, endpoint, input, output)
}

//----------------------------------------------------------

// Use these when doing HTTP requests where the inputs and outputs
// are supposed to be JSON strings, and the output is in the form
// of a JsonResponse

func (h *Http) PzGet(endpoint string) *JsonResponse {
	output := &JsonResponse{}

	code, err := h.Get(endpoint, output)
	if err != nil {
		return newJsonResponse500(err)
	}

	output.StatusCode = code

	return output
}

func (h *Http) PzGet2(endpoint string, input interface{}) *JsonResponse {
	output := &JsonResponse{}

	code, err := h.Get2(endpoint, input, output)
	if err != nil {
		return newJsonResponse500(err)
	}

	output.StatusCode = code

	return output
}

func (h *Http) postOrPut(post bool, endpoint string, input interface{}) *JsonResponse {
	output := &JsonResponse{}

	f := h.Post
	if !post {
		f = h.Put
	}

	code, err := f(endpoint, input, output)
	if err != nil {
		return newJsonResponse500(err)
	}

	output.StatusCode = code

	return output
}

func (h *Http) PzPost(endpoint string, input interface{}) *JsonResponse {
	return h.postOrPut(true, endpoint, input)
}

func (h *Http) PzPut(endpoint string, input interface{}) *JsonResponse {
	return h.postOrPut(false, endpoint, input)
}

func (h *Http) PzDelete(endpoint string) *JsonResponse {
	output := &JsonResponse{}
	code, err := h.Delete(endpoint, output)
	if err != nil {
		return newJsonResponse500(err)
	}
	output.StatusCode = code
	return output
}

//---------------------------------------------------------------------

type SimpleFlightCheck struct {
	NumPreflights  int
	NumPostflights int
}

func (fc *SimpleFlightCheck) Preflight(verb string, url string, obj string) error {
	log.Printf("PREFLIGHT.verb: %s", verb)
	log.Printf("PREFLIGHT.url: %s", url)
	log.Printf("PREFLIGHT.obj: %s", obj)
	fc.NumPreflights++
	return nil
}

func (fc *SimpleFlightCheck) Postflight(code int, obj string) error {
	log.Printf("POSTFLIGHT.code: %d", code)
	log.Printf("POSTFLIGHT.obj: %s", obj)
	fc.NumPostflights++
	return nil
}
