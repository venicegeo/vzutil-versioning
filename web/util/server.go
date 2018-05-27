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

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/braintree/manners"
	"github.com/gin-gonic/gin"
	g "github.com/venicegeo/pz-gocommon/gocommon"
)

type Server struct {
	router           http.Handler
	obj              *manners.GracefulServer
	authRedirectPath string
	certFile         string
	keyFile          string
	authCollection   map[string]authInfo
	authTimeout      time.Duration
}

type RouteData struct {
	Verb         string
	Path         string
	Handler      gin.HandlerFunc
	RequiresAuth bool
}
type authInfo struct {
	authorizedUntil time.Time
	remoteAddr      string
}

func NewServer() *Server {
	return &Server{nil, nil, "/login", "", "", map[string]authInfo{}, time.Minute * 15}
}

func (server *Server) SetAuthRedirectPath(path string) {
	server.authRedirectPath = path
}
func (server *Server) SetAuthTimeout(dur time.Duration) {
	server.authTimeout = dur
}
func (server *Server) SetTLSInfo(certFile, keyFile string) {
	server.certFile = certFile
	server.keyFile = keyFile
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
		if server.certFile != "" && server.keyFile != "" {
			done <- server.obj.ListenAndServeTLS(server.certFile, server.keyFile)
		} else {
			done <- server.obj.ListenAndServe()
		}
	}()

	return done
}
func (server *Server) Configure(routeData []RouteData) error {
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()

	for _, data := range routeData {
		function := server.middleware(data)
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

	router.LoadHTMLGlob("templates/*")
	server.router = router

	return nil
}

func (server *Server) middleware(route RouteData) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("_method", route.Verb)
		c.Set("_route", route.Path)
		if route.RequiresAuth {
			if auth, err := server.VerifyAuth(c); err != nil {
				c.String(400, "Unknown error with auth")
				return
			} else if !auth {
				fmt.Println("redirecting to auth")
				c.Redirect(303, server.authRedirectPath)
				return
			}
		}
		route.Handler(c)
		c.Next()
	}
}

func (server *Server) VerifyAuth(c *gin.Context) (bool, error) {
	cookie, err := c.Request.Cookie("auth")
	if err == http.ErrNoCookie {
		fmt.Println("No cookie")
		return false, nil
	} else if err != nil {
		return false, err
	}
	auth, ok := server.authCollection[cookie.Value]
	if !ok {
		fmt.Println("not found locally")
		return false, nil
	}
	if time.Now().After(auth.authorizedUntil) {
		fmt.Println(time.Now(), auth.authorizedUntil)
		delete(server.authCollection, cookie.Value)
		return false, nil
	} else if strings.SplitN(c.Request.RemoteAddr, ":", 2)[0] != auth.remoteAddr {
		fmt.Println(strings.SplitN(c.Request.RemoteAddr, ":", 2)[0], auth.remoteAddr)
		return false, nil
	}
	return true, nil
}

func (server *Server) CreateAuth(c *gin.Context) {
	expires := time.Now().Add(server.authTimeout)
	fmt.Println("Setting expired to", expires)
	key := g.NewUuid().String()
	for {
		if _, ok := server.authCollection[key]; ok {
			key = g.NewUuid().String()
		} else {
			break
		}
	}
	server.authCollection[key] = authInfo{expires, strings.SplitN(c.Request.RemoteAddr, ":", 2)[0]}
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "auth",
		Value:    key,
		Expires:  expires,
		Domain:   os.Getenv("DOMAIN"),
		HttpOnly: true,
		Secure:   true,
	})
}
