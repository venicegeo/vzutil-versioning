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

package syslog

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/venicegeo/pz-gocommon/elasticsearch"
	"github.com/venicegeo/pz-gocommon/gocommon"
)

//---------------------------------------------------------------------

func fileEquals(t *testing.T, expected string, fileName string) {
	assert := assert.New(t)

	buf, err := ioutil.ReadFile(fileName)
	assert.NoError(err)

	assert.EqualValues(expected, string(buf))
}

func fileExist(s string) bool {
	if _, err := os.Stat(s); os.IsNotExist(err) {
		return false
	}
	return true
}

func safeRemove(s string) error {
	if fileExist(s) {
		err := os.Remove(s)
		if err != nil {
			return err
		}
	}
	return nil
}

func makeMessage(sde bool) (*Message, string) {
	m := NewMessage("123456")

	m.Facility = DefaultFacility
	m.Severity = Fatal // pri = 1*8 + 2 = 10
	m.Version = DefaultVersion
	m.TimeStamp = piazza.NewTimeStamp()
	m.HostName = "HOST"
	m.Application = "APPLICATION"
	m.Process = "1234"
	m.MessageID = "msg1of2"
	m.AuditData = nil
	m.MetricData = nil
	m.Message = "AlphaYow"

	expected := "<10>1 " + m.TimeStamp.String() + " HOST APPLICATION 1234 msg1of2 - AlphaYow"

	if sde {
		m.Message = "BetaYow"

		m.AuditData = &AuditElement{
			Actor:  "=actor=",
			Action: "-action-",
			Actee:  "_actee_",
		}
		m.MetricData = &MetricElement{
			Name:   "=name=",
			Value:  -3.14,
			Object: "_object_",
		}

		expected = "<10>1 " + m.TimeStamp.String() + " HOST APPLICATION 1234 msg1of2 " +
			"[pzaudit@123456 actor=\"=actor=\" action=\"-action-\" actee=\"_actee_\"] " +
			"[pzmetric@123456 name=\"=name=\" value=\"-3.140000\" object=\"_object_\"] " +
			"BetaYow"
	}

	return m, expected
}

func simpleChecker(t *testing.T, m *Message, severity Severity, text string) {
	assert := assert.New(t)

	facility := DefaultFacility
	host, err := os.Hostname()
	assert.NoError(err)
	pid := fmt.Sprintf("%d", os.Getpid())

	assert.EqualValues(facility, m.Facility)
	assert.EqualValues(severity, m.Severity)
	assert.EqualValues(pid, m.Process)
	assert.EqualValues(host, m.HostName)
	assert.EqualValues(text, m.Message)

	err = m.Validate()
	assert.NoError(err)

	s := m.String()
	assert.NotEmpty(s)
}

//---------------------------------------------------------------------

func Test01Message(t *testing.T) {
	assert := assert.New(t)

	m, expected := makeMessage(false)

	s := m.String()
	assert.EqualValues(expected, s)

	err := m.Validate()
	assert.NoError(err)

	//	mm, err := ParseMessageString(expected)
	//	assert.NoError(err)

	//	assert.EqualValues(m, mm)
}

func Test02MessageSDE(t *testing.T) {
	assert := assert.New(t)

	m, expected := makeMessage(true)

	s := m.String()
	assert.EqualValues(expected, s)

	err := m.Validate()
	assert.NoError(err)

	// TODO: this won't work until we make the parser understand SDEs
	//mm, err := ParseMessageString(expected)
	//assert.NoError(err)
	//assert.EqualValues(m, mm)
}

func Test03LocalWriter(t *testing.T) {
	assert := assert.New(t)

	mssg1, _ := makeMessage(false)
	mssg2, _ := makeMessage(false)

	w := &LocalReaderWriter{}

	actual, err := w.Read(1)
	assert.NoError(err)
	assert.Len(actual, 0)

	err = w.Write(mssg1, false)
	assert.NoError(err)

	actual, err = w.Read(0)
	assert.NoError(err)
	assert.Len(actual, 0)

	actual, err = w.Read(1)
	assert.NoError(err)
	assert.Len(actual, 1)
	assert.EqualValues(*mssg1, actual[0])

	actual, err = w.Read(2)
	assert.NoError(err)
	assert.Len(actual, 1)
	assert.EqualValues(*mssg1, actual[0])

	err = w.Write(mssg2, false)
	assert.NoError(err)

	actual, err = w.Read(2)
	assert.NoError(err)
	assert.Len(actual, 2)
	assert.EqualValues(*mssg1, actual[0])
	assert.EqualValues(*mssg2, actual[1])

	_, err = w.Read(-9)
	assert.Error(err)

	_, _, err = w.GetMessages(nil, nil)
	assert.Error(err)
	err = w.Close()
	assert.NoError(err)
}

func Test04FileWriter(t *testing.T) {
	var err error

	assert := assert.New(t)

	fname := "./testsyslog.txt"

	err = safeRemove(fname)
	assert.NoError(err)

	m1, expected1 := makeMessage(false)
	m2, expected2 := makeMessage(true)
	{
		w := &FileWriter{FileName: fname}
		err = w.Write(m1, false)
		assert.NoError(err)
		err = w.Close()
		assert.NoError(err)

		fileEquals(t, expected1+"\n", fname)
	}

	{
		w := &FileWriter{FileName: fname}
		err = w.Write(m2, false)
		assert.NoError(err)
		err = w.Close()
		assert.NoError(err)
		fileEquals(t, expected1+"\n"+expected2+"\n", fname)
	}

	err = safeRemove(fname)
	assert.NoError(err)
}

func Test05Logger(t *testing.T) {
	assert := assert.New(t)
	var err error

	logWriter := &LocalReaderWriter{}
	auditWriter := &LocalReaderWriter{}

	// the following clause is what a developer would do
	{
		logger := NewLogger(logWriter, auditWriter, "testapp", "123456")
		logger.Async = false
		logger.UseSourceElement = false
		err = logger.Debug("debug %d", 999)
		assert.NoError(err)
		err = logger.Info("info %d", 123)
		assert.NoError(err)
		err = logger.Notice("notice %d", 321)
		assert.NoError(err)
		err = logger.Warn("bonk %d", 3)
		assert.NoError(err)
		err = logger.Error("Bonk %s", ".1")
		assert.NoError(err)
		err = logger.Fatal("BONK %f", 4.0)
		assert.NoError(err)
		err = logger.Audit("1", "2", "3", "brown%s", "fox")
		assert.NoError(err)
		err = logger.Metric("i", 5952567, "k", "lazy%s", "dog")
		assert.NoError(err)

		err = logger.Information("Info/Information")
		assert.NoError(err)
		err = logger.Warning("Warn/Warning")
		assert.NoError(err)
	}

	mssgs, err := logWriter.Read(100)
	assert.NoError(err)
	assert.Len(mssgs, 10)

	simpleChecker(t, &mssgs[0], Debug, "debug 999")
	simpleChecker(t, &mssgs[1], Informational, "info 123")
	simpleChecker(t, &mssgs[2], Notice, "notice 321")
	simpleChecker(t, &mssgs[3], Warning, "bonk 3")
	simpleChecker(t, &mssgs[4], Error, "Bonk .1")
	simpleChecker(t, &mssgs[5], Fatal, "BONK 4.000000")
	simpleChecker(t, &mssgs[6], Notice, "brownfox")
	assert.EqualValues("1", mssgs[6].AuditData.Actor)
	assert.EqualValues("2", mssgs[6].AuditData.Action)
	assert.EqualValues("3", mssgs[6].AuditData.Actee)
	simpleChecker(t, &mssgs[7], Notice, "lazydog")
	assert.EqualValues("i", mssgs[7].MetricData.Name)
	assert.EqualValues(5952567, mssgs[7].MetricData.Value)
	assert.EqualValues("k", mssgs[7].MetricData.Object)

	simpleChecker(t, &mssgs[8], Informational, "Info/Information")
	simpleChecker(t, &mssgs[9], Warning, "Warn/Warning")
}

func Test06LogLevel(t *testing.T) {
	assert := assert.New(t)
	var err error

	logWriter := &LocalReaderWriter{}
	auditWriter := &LocalReaderWriter{}

	{
		logger := NewLogger(logWriter, auditWriter, "testapp", "123456")
		logger.Async = false
		logger.UseSourceElement = true
		logger.MinimumSeverity = Error
		err = logger.Warning("bonk")
		assert.NoError(err)
		err = logger.Error("Bonk")
		assert.NoError(err)
		err = logger.Fatal("BONK")
		assert.NoError(err)
	}

	mssgs, err := logWriter.Read(10)
	assert.NoError(err)
	assert.Len(mssgs, 2)

	simpleChecker(t, &mssgs[0], Error, "Bonk")
	simpleChecker(t, &mssgs[1], Fatal, "BONK")
}

func Test07StackFrame(t *testing.T) {
	assert := assert.New(t)

	function, file, line := stackFrame(-1)
	//log.Printf("%s\t%s\t%d", function, file, line)
	assert.EqualValues(file, "Message.go")
	assert.True(line > 1 && line < 1000)
	assert.EqualValues("syslog.stackFrame", function)

	function, file, line = stackFrame(0)
	//log.Printf("%s\t%s\t%d", function, file, line)
	assert.EqualValues(file, "Syslog_test.go")
	assert.True(line > 1 && line < 1000)
	assert.EqualValues("syslog.Test07StackFrame", function)
}

func Test08SyslogdWriter(t *testing.T) {
	assert := assert.New(t)

	m1, _ := makeMessage(false)
	m2, _ := makeMessage(true)

	w := &SyslogdWriter{}

	err := w.Write(m1, false)
	assert.NoError(err)
	assert.Equal(1, w.writer.numWritten)

	err = w.Write(m2, false)
	assert.NoError(err)
	assert.Equal(2, w.writer.numWritten)

	// TODO: how can we check the syslogd system got our messages?

	err = w.Close()
	assert.NoError(err)
}

func Test09ElasticsearchWriter(t *testing.T) {
	assert := assert.New(t)
	var err error

	esi := elasticsearch.NewMockIndex("test09")
	err = esi.Create("")
	assert.NoError(err)

	ew := &ElasticWriter{Esi: esi}
	assert.NotNil(ew)
	err = ew.SetType("Baz")
	assert.NoError(err)

	ew2 := NewElasticWriter(esi, "Baz")
	assert.NotNil(ew2)
	assert.EqualValues(ew, ew2)
	err = ew2.SetID("foobarbaz")
	assert.NoError(err)
	_, err = ew2.Read(1)
	assert.Error(err)
	err = ew2.Close()
	assert.NoError(err)

	m := NewMessage("123456")
	m.Message = "Yow"
	err = ew.Write(m, false)
	assert.NoError(err)

	params := &piazza.HttpQueryParams{}
	format, err := piazza.NewJsonPagination(params)
	assert.NoError(err)
	x, err := esi.FilterByMatchAll("", format)
	assert.NoError(err)

	assert.Len(*x.GetHits(), 1)

	src := x.GetHit(0).Source
	assert.NotNil(src)
	var tmp1 Message
	err = json.Unmarshal(*src, &tmp1)
	assert.NoError(err)
	assert.EqualValues("Yow", tmp1.Message)
}

func Test10Errors(t *testing.T) {
	assert := assert.New(t)

	logger := NewLogger(nil, nil, "testapp", "123456")
	logger.Async = false
	err := logger.Warning("bonk")
	assert.Error(err)
}

//---------------------------------------------------------------------

type TThingServer struct {
	routes   []piazza.RouteData
	lastPost interface{}
	lastUrl  string
}

func (server *TThingServer) Init() {
	server.routes = []piazza.RouteData{
		{Verb: "GET", Path: "/", Handler: server.handleGetRoot},
		{Verb: "GET", Path: "/admin/stats", Handler: server.handleGetStats},
		{Verb: "GET", Path: "/version", Handler: server.handleGetVersion},
		{Verb: "GET", Path: "/syslog", Handler: server.handleGet},
		{Verb: "POST", Path: "/syslog", Handler: server.handlePost},
	}
}

func (server *TThingServer) handleGetRoot(c *gin.Context) {
	resp := &piazza.JsonResponse{
		StatusCode: http.StatusOK,
	}
	piazza.GinReturnJson(c, resp)
}

func (server *TThingServer) handleGetVersion(c *gin.Context) {
	version := "1.2.3.4"

	resp := &piazza.JsonResponse{
		StatusCode: http.StatusOK,
		Data:       piazza.Version{Version: version},
	}
	piazza.GinReturnJson(c, resp)
}

type TThingStats struct {
	Count int
}

func (server *TThingServer) handleGetStats(c *gin.Context) {
	stats := &TThingStats{Count: 19}
	resp := &piazza.JsonResponse{
		StatusCode: http.StatusOK,
		Data:       stats,
	}
	piazza.GinReturnJson(c, resp)
}

func (server *TThingServer) handleGet(c *gin.Context) {
	server.lastPost = nil
	server.lastUrl = c.Request.URL.String()

	m1, _ := makeMessage(false)
	m2, _ := makeMessage(true)

	resp := &piazza.JsonResponse{
		StatusCode: http.StatusOK,
		Pagination: &piazza.JsonPagination{Count: 17},
		Data:       &[]Message{*m1, *m2},
	}

	piazza.GinReturnJson(c, resp)
}

func (server *TThingServer) handlePost(c *gin.Context) {
	var thing interface{}
	err := c.BindJSON(&thing)
	if err != nil {
		resp := &piazza.JsonResponse{StatusCode: http.StatusInternalServerError, Message: err.Error()}
		piazza.GinReturnJson(c, resp)
	}
	resp := &piazza.JsonResponse{
		StatusCode: http.StatusOK,
	}
	server.lastPost = thing
	server.lastUrl = c.Request.URL.String()
	piazza.GinReturnJson(c, resp)
}

func Test11HttpWriter(t *testing.T) {
	assert := assert.New(t)

	var ts *TThingServer
	var gs *piazza.GenericServer
	var w Writer

	//w.h.FlightCheck = &piazza.SimpleFlightCheck{}

	{
		required := []piazza.ServiceName{}
		sys, err := piazza.NewSystemConfig(piazza.PzGoCommon, required)
		assert.NoError(err)

		gs = &piazza.GenericServer{Sys: sys}

		ts = &TThingServer{}
		ts.Init()

		err = gs.Configure(ts.routes)
		assert.NoError(err)
		_, err = gs.Start()
		assert.NoError(err)

		//apiServer, err := piazza.GetApiServer()
		//assert.NoError(err)
		//apiKey, err := piazza.GetApiKey(apiServer)
		//assert.NoError(err)

		w, err = NewHttpWriter("http://"+sys.BindTo, "")
		assert.NoError(err)
	}

	// test writing
	{
		m1, _ := makeMessage(false)
		m2, _ := makeMessage(true)

		err := w.Write(m1, false)
		assert.NoError(err)
		assert.EqualValues("/syslog", ts.lastUrl)
		p1a := ts.lastPost.(map[string]interface{})
		assert.EqualValues("AlphaYow", p1a["message"])
		p1b := ts.lastPost.(map[string]interface{})
		assert.Nil(p1b["auditData"])

		err = w.Write(m2, false)
		assert.NoError(err)
		assert.EqualValues("/syslog", ts.lastUrl)
		p2a := ts.lastPost.(map[string]interface{})
		assert.EqualValues("BetaYow", p2a["message"])
		p2b := ts.lastPost.(map[string]interface{})
		assert.NotNil(p2b["auditData"])
	}

	// test reading
	{
		jpage := &piazza.JsonPagination{
			Page:    2,
			PerPage: 4,
			SortBy:  "frozz",
			Order:   "desc",
		}
		params := &piazza.HttpQueryParams{}

		ww := w.(*HttpWriter)
		assert.NotNil(ww)

		{
			mssgs, count, err := ww.GetMessages(jpage, params)
			assert.NoError(err)
			assert.EqualValues("AlphaYow", mssgs[0].Message)
			assert.EqualValues("BetaYow", mssgs[1].Message)

			assert.EqualValues("/syslog?perPage=4&page=2&sortBy=frozz&order=desc", ts.lastUrl)
			assert.NotNil(mssgs)
			assert.Len(mssgs, 2)
			assert.Equal(17, count)
		}
		{
			ms, err := ww.Read(2)
			assert.NoError(err)
			assert.EqualValues("AlphaYow", ms[0].Message)
			assert.EqualValues("BetaYow", ms[1].Message)
		}
	}

	{
		err := w.Close()
		assert.NoError(err)

		err = gs.Stop()
		assert.NoError(err)
	}
}

func Test12NilWriter(t *testing.T) {
	assert := assert.New(t)
	w := NilWriter{}
	m1, _ := makeMessage(false)
	err := w.Write(m1, false)
	assert.NoError(err)
	err = w.Close()
	assert.NoError(err)
}

func Test13MultiWriter(t *testing.T) {
	assert := assert.New(t)

	var err error

	m1, _ := makeMessage(false)
	m2, _ := makeMessage(true)

	w1 := &LocalReaderWriter{}
	w2 := &LocalReaderWriter{}
	ws := []Writer{w1, w2}

	mw := NewMultiWriter(ws)

	err = mw.Write(m1, false)
	assert.NoError(err)

	err = mw.Write(m2, false)
	assert.NoError(err)

	{
		ms, err := w1.Read(2)
		assert.NoError(err)
		assert.EqualValues("AlphaYow", ms[0].Message)
		assert.EqualValues("BetaYow", ms[1].Message)
	}
	{
		ms, err := w2.Read(2)
		assert.NoError(err)
		assert.EqualValues("AlphaYow", ms[0].Message)
		assert.EqualValues("BetaYow", ms[1].Message)
	}

	err = mw.Close()
	assert.NoError(err)
}
