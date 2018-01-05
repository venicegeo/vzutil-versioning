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
	"errors"
	"net/http"
	"strings"

	"github.com/braintree/manners"
	"github.com/gin-gonic/gin"
)

type Server struct {
	router http.Handler
	obj    *manners.GracefulServer
}

type RouteData struct {
	Verb    string
	Path    string
	Handler gin.HandlerFunc
}

func (server *Server) Stop() error {
	server.obj.Close()
	return nil
}
func (server *Server) Start(uri string) chan error {
	done := make(chan error)
	server.obj = manners.NewWithServer(&http.Server{
		Addr:    uri,
		Handler: server.router,
	})

	go func() {
		err := server.obj.ListenAndServe()
		done <- err
	}()

	return done
}
func (server *Server) Configure(routeData []RouteData) error {
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()

	for _, data := range routeData {
		function := addData(data.Verb, strings.TrimPrefix(data.Path, "/"), data.Handler)
		switch data.Verb {
		case "GET":
			router.GET(data.Path, function)
		case "POST":
			router.POST(data.Path, function)
		case "PUT":
			router.PUT(data.Path, function)
		case "DELETE":
			router.DELETE(data.Path, function)
		default:
			return errors.New("Invalid verb: " + data.Verb)
		}
	}

	server.router = router

	return nil
}

func addData(method, route string, handler gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("_method", method)
		c.Set("_route", route)
		handler(c)
		c.Next()
	}
}
