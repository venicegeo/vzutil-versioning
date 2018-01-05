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
	"fmt"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

//--------------------------

type Thing struct {
	ID    string `json:"id"`
	Value string `json:"value"`
}

type ThingService struct {
	assert  *assert.Assertions
	Data    map[string]string `json:"data"`
	IDCount int
}

func (service *ThingService) GetThing(id string) *JsonResponse {
	val, ok := service.Data[id]
	if !ok {
		return &JsonResponse{StatusCode: http.StatusNotFound}
	}
	resp := &JsonResponse{StatusCode: http.StatusOK, Data: Thing{ID: id, Value: val}}
	err := resp.SetType()
	if err != nil {
		return &JsonResponse{StatusCode: http.StatusInternalServerError, Message: err.Error()}
	}
	return resp
}

func (service *ThingService) PostThing(thing *Thing) *JsonResponse {
	if thing.Value == "NULL" {
		resp := &JsonResponse{StatusCode: http.StatusBadRequest, Message: "oops"}
		return resp
	}
	service.IDCount++
	thing.ID = fmt.Sprintf("%d", service.IDCount)
	service.Data[thing.ID] = thing.Value

	resp := &JsonResponse{StatusCode: http.StatusCreated, Data: thing}
	err := resp.SetType()
	if err != nil {
		return &JsonResponse{StatusCode: http.StatusInternalServerError, Message: err.Error()}
	}
	return resp
}

func (service *ThingService) PutThing(id string, thing *Thing) *JsonResponse {
	if thing.Value == "NULL" {
		return &JsonResponse{StatusCode: http.StatusBadRequest, Message: "oops"}
	}
	if thing.ID != id {
		return &JsonResponse{StatusCode: http.StatusBadRequest, Message: "oops - id mismatch"}
	}
	service.Data[thing.ID] = thing.Value

	resp := &JsonResponse{StatusCode: http.StatusOK, Data: thing}
	err := resp.SetType()
	if err != nil {
		return &JsonResponse{StatusCode: http.StatusInternalServerError, Message: err.Error()}
	}
	return resp
}

func (service *ThingService) DeleteThing(id string) *JsonResponse {
	_, ok := service.Data[id]
	if !ok {
		return &JsonResponse{StatusCode: http.StatusNotFound}
	}
	delete(service.Data, id)
	return &JsonResponse{StatusCode: http.StatusOK}
}

//---------------------------------------------------------------

type ThingServer struct {
	routes  []RouteData
	service *ThingService
}

func (server *ThingServer) Init(service *ThingService) {

	server.service = service

	server.routes = []RouteData{
		{"GET", "/", server.handleGetRoot},
		{"GET", "/:id", server.handleGet},
		{"POST", "/", server.handlePost},
		{"PUT", "/:id", server.handlePut},
		{"DELETE", "/:id", server.handleDelete},
	}
}

func (server *ThingServer) handleGetRoot(c *gin.Context) {
	type T struct {
		Message string
	}
	message := "Hi."
	resp := JsonResponse{StatusCode: http.StatusOK, Data: message}
	err := resp.SetType()
	if err != nil {
		resp = JsonResponse{StatusCode: http.StatusInternalServerError, Message: err.Error()}
	}
	GinReturnJson(c, &resp)
}

func (server *ThingServer) handleGet(c *gin.Context) {
	id := c.Param("id")
	resp := server.service.GetThing(id)
	GinReturnJson(c, resp)
}

func (server *ThingServer) handlePost(c *gin.Context) {
	var thing Thing
	err := c.BindJSON(&thing)
	if err != nil {
		resp := &JsonResponse{StatusCode: http.StatusInternalServerError, Message: err.Error()}
		GinReturnJson(c, resp)
	}
	resp := server.service.PostThing(&thing)
	GinReturnJson(c, resp)
}

func (server *ThingServer) handlePut(c *gin.Context) {
	id := c.Param("id")
	var thing Thing
	err := c.BindJSON(&thing)
	if err != nil {
		resp := &JsonResponse{StatusCode: http.StatusInternalServerError, Message: err.Error()}
		GinReturnJson(c, resp)
	}
	thing.ID = id
	resp := server.service.PutThing(id, &thing)
	GinReturnJson(c, resp)
}

func (server *ThingServer) handleDelete(c *gin.Context) {
	id := c.Param("id")
	resp := server.service.DeleteThing(id)
	GinReturnJson(c, resp)
}

//------------------------------------------

func Test07Server(t *testing.T) {
	assert := assert.New(t)

	JsonResponseDataTypes["string"] = "string"
	JsonResponseDataTypes["piazza.Thing"] = "thing"
	JsonResponseDataTypes["*piazza.Thing"] = "thing"

	var genericServer *GenericServer
	var server *ThingServer
	var sys *SystemConfig

	{
		var err error
		required := []ServiceName{}
		sys, err = NewSystemConfig(PzGoCommon, required)
		assert.NoError(err)

		genericServer = &GenericServer{Sys: sys}
		server = &ThingServer{}
		service := &ThingService{
			assert:  assert,
			IDCount: 0,
			Data:    make(map[string]string),
		}

		server.Init(service)
	}

	h := &Http{}

	{
		var err error
		err = genericServer.Configure(server.routes)
		if err != nil {
			assert.FailNow("server failed to configure: " + err.Error())
		}
		_, err = genericServer.Start()
		if err != nil {
			assert.FailNow("server failed to start: " + err.Error())
		}

		h.BaseUrl = "http://" + sys.BindTo
	}

	var input *Thing
	var output Thing
	var jresp *JsonResponse = &JsonResponse{}

	{
		var err error

		// GET /
		jresp = h.PzGet("/")
		assert.Equal(200, jresp.StatusCode)
		assert.EqualValues("string", jresp.Type)

		// GET bad
		jresp = h.PzGet("/mpg")
		assert.Equal(404, jresp.StatusCode)

		// POST 1
		input = &Thing{Value: "17"}
		jresp = h.PzPost("/", input)
		assert.Equal(201, jresp.StatusCode)
		assert.EqualValues("thing", jresp.Type)

		err = jresp.ExtractData(&output)
		assert.NoError(err)
		assert.EqualValues("1", output.ID)
		assert.EqualValues("17", output.Value)

		// POST bad
		input = &Thing{Value: "NULL"}
		jresp = h.PzPost("/", input)
		assert.Equal(400, jresp.StatusCode)

		// POST 2
		input = &Thing{Value: "18"}
		jresp = h.PzPost("/", input)
		assert.Equal(201, jresp.StatusCode)
		assert.EqualValues("thing", jresp.Type)

		err = jresp.ExtractData(&output)
		assert.NoError(err)
		assert.EqualValues("2", output.ID)
		assert.EqualValues("18", output.Value)

		// GET 2
		jresp = h.PzGet("/2")
		assert.Equal(200, jresp.StatusCode)
		assert.EqualValues("thing", jresp.Type)

		err = jresp.ExtractData(&output)
		assert.NoError(err)
		assert.EqualValues("2", output.ID)
		assert.EqualValues("18", output.Value)

		// PUT 1
		input = &Thing{Value: "71"}
		jresp = h.PzPut("/1", input)
		assert.Equal(200, jresp.StatusCode)
		assert.EqualValues("thing", jresp.Type)

		err = jresp.ExtractData(&output)
		assert.NoError(err)
		assert.EqualValues("71", output.Value)

		// GET 1
		jresp = h.PzGet("/1")
		assert.Equal(200, jresp.StatusCode)
		assert.EqualValues("thing", jresp.Type)

		err = jresp.ExtractData(&output)
		assert.NoError(err)
		assert.EqualValues("1", output.ID)
		assert.EqualValues("71", output.Value)

		// DELETE 3
		jresp = h.PzDelete("/3")
		assert.Equal(404, jresp.StatusCode)

		// DELETE 1
		jresp = h.PzDelete("/1")
		assert.Equal(200, jresp.StatusCode)

		// GET 1
		jresp = h.PzGet("/1")
		assert.Equal(404, jresp.StatusCode)
	}

	// raw PUT and DELETE
	{
		// PUT
		input = &Thing{Value: "72"}
		body, err := h.convertObjectToReader(input)
		assert.NoError(err)
		resp, err := HTTPPut(h.BaseUrl+"/2", ContentTypeJSON, body)
		assert.NoError(err)
		assert.Equal(200, resp.StatusCode)

		// check return
		err = h.convertResponseBodyToObject(resp, jresp)
		assert.NoError(err)
		m := jresp.Data.(map[string]interface{})
		assert.Equal("72", m["value"])

		// DELETE
		resp, err = HTTPDelete(h.BaseUrl + "/2")
		assert.NoError(err)
		assert.Equal(200, resp.StatusCode)

		// check return
		// GET 2
		jresp = h.PzGet("/2")
		assert.Equal(404, jresp.StatusCode)
	}

	{
		_ = genericServer.Stop()
		//assert.NoError(err)

		_, err := http.Get(h.BaseUrl)
		assert.Error(err)
	}
}

func Test07ABadServer(t *testing.T) {
	assert := assert.New(t)

	f := func(c *gin.Context) {}

	required := []ServiceName{}
	sys, err := NewSystemConfig(PzGoCommon, required)
	assert.NoError(err)

	genericServer := GenericServer{Sys: sys}

	service := &ThingService{
		assert:  assert,
		IDCount: 0,
		Data:    make(map[string]string),
	}

	server := &ThingServer{}

	server.Init(service)

	server.routes = []RouteData{
		{"GET", "/", f},
		{"YOW", "/", f},
	}

	err = genericServer.Configure(server.routes)
	assert.Error(err)
}
