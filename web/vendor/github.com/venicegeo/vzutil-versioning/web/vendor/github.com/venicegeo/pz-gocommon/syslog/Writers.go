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
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"reflect"

	"github.com/venicegeo/pz-gocommon/elasticsearch"
	piazza "github.com/venicegeo/pz-gocommon/gocommon"
)

const (
	SyslogdNetwork = ""
	SyslogdRaddr   = ""
	LoggerType     = "LogData"
)

func GetRequiredEnvVars() (string, error) {
	var loggerIndex string
	if loggerIndex = os.Getenv("LOGGER_INDEX"); loggerIndex == "" {
		return "", errors.New("Elasticsearch index name not set")
	}
	return loggerIndex, nil
}

func GetRequiredWriters(sys *piazza.SystemConfig, loggerIndex string) (Writer, Writer, error) {
	var indexExists bool
	var err error
	indexExists, err = elasticsearch.IndexExists(sys, loggerIndex)
	if err != nil {
		return nil, nil, err
	}
	if !indexExists {
		return &StderrWriter{}, &StdoutWriter{}, nil
	}
	esi, err := elasticsearch.NewIndex(sys, loggerIndex, "")
	if err != nil {
		return nil, nil, err
	}
	logWriter := &ElasticWriter{Esi: esi}
	if err = logWriter.SetType(LoggerType); err != nil {
		return nil, nil, err
	}
	return logWriter, &StdoutWriter{}, err
}

//---------------------------------------------------------------------

// Writer is an interface for writing a Message to some sort of output.
type Writer interface {
	Write(*Message, bool) error
	writeWork(*Message) error
	Close() error
}

// Reader is an interface for reading Messages from some sort of input.
// count is the number of messages to read: 1 means the latest message,
// 2 means the two latest messages, etc. The newest message is at the end
// of the array.
type Reader interface {
	Read(count int) ([]Message, error)
	GetMessages(*piazza.JsonPagination, *piazza.HttpQueryParams) ([]Message, int, error)
}

type WriterReader interface {
	Reader
	Writer
}

//---------------------------------------------------------------------

type writeWork func(*Message) error

func aSyncLogic(write writeWork, mssg *Message, async bool) (err error) {
	if async {
		go func() {
			if err = write(mssg); err != nil {
				log.Printf("Unable to log message [%s] : %s\n", mssg.String(), err.Error())
			}
		}()
		return nil
	}
	if err = write(mssg); err != nil {
		log.Printf("Unable to log message [%s] : %s\n", mssg.String(), err.Error())
	}
	return err
}

//---------------------------------------------------------------------

// FileWriter implements the Writer interface, writing to a given file
type FileWriter struct {
	FileName string
	file     *os.File
}

// Write writes the message to the supplied file.
func (w *FileWriter) Write(mssg *Message, async bool) error {
	var _ Writer = (*FileWriter)(nil)
	return w.writeWork(mssg)
}

func (w *FileWriter) writeWork(mssg *Message) (err error) {
	if w == nil || w.FileName == "" {
		return fmt.Errorf("writer not set not set")
	}

	if w.file == nil {
		w.file, err = os.OpenFile(w.FileName, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
		if err != nil {
			return err
		}
	}

	s := mssg.String()
	s += "\n"

	_, err = io.WriteString(w.file, s)
	return err
}

// Close closes the file. The creator of the FileWriter must call this.
func (w *FileWriter) Close() error {
	return w.file.Close()
}

//---------------------------------------------------------------------

//StdoutWriter writes messages to STDOUT
type StdoutWriter struct {
}

//Writes message to STDOUT
func (w *StdoutWriter) Write(mssg *Message, async bool) error {
	var _ Writer = (*StdoutWriter)(nil)
	return w.writeWork(mssg)
}

func (w *StdoutWriter) writeWork(mssg *Message) error {
	fmt.Println(mssg.String())
	return nil
}

//Nothing to close for this writer
func (w *StdoutWriter) Close() error {
	return nil
}

//---------------------------------------------------------------------

//StderrWriter writes messages to STDERR
type StderrWriter struct {
}

//Writes message to STDERR
func (w *StderrWriter) Write(mssg *Message, async bool) error {
	var _ Writer = (*StderrWriter)(nil)
	return w.writeWork(mssg)
}

func (w *StderrWriter) writeWork(mssg *Message) error {
	fmt.Fprintln(os.Stderr, mssg.String())
	return nil
}

//Nothing to close for this writer
func (w *StderrWriter) Close() error {
	return nil
}

//---------------------------------------------------------------------

// MessageWriter implements Reader and Writer, using an array of Messages
// as the backing store
type LocalReaderWriter struct {
	messages []Message
}

// Write writes the message to the backing array
func (w *LocalReaderWriter) Write(mssg *Message, async bool) error {
	var _ Writer = (*LocalReaderWriter)(nil)
	return w.writeWork(mssg)
}

func (w *LocalReaderWriter) writeWork(mssg *Message) error {
	if w.messages == nil {
		w.messages = make([]Message, 0)
	}
	w.messages = append(w.messages, *mssg)
	return nil
}

// Read reads messages from the backing array. Will only return as many as are
// available; asking for too many is not an error.
func (w *LocalReaderWriter) Read(count int) ([]Message, error) {
	var _ Reader = (*LocalReaderWriter)(nil)

	if count < 0 {
		return nil, fmt.Errorf("invalid count: %d", count)
	}

	if w.messages == nil || count == 0 {
		return make([]Message, 0), nil
	}

	if count > len(w.messages) {
		count = len(w.messages)
	}

	n := len(w.messages)
	a := w.messages[n-count : n]

	return a, nil
}

func (w *LocalReaderWriter) GetMessages(*piazza.JsonPagination, *piazza.HttpQueryParams) ([]Message, int, error) {
	return nil, 0, fmt.Errorf("LocalReaderWriter.GetMessages() not supported")
}

func (w *LocalReaderWriter) Close() error {
	return nil
}

//---------------------------------------------------------------------

// HttpWriter implements Writer, by talking to the actual pz-logger service
type HttpWriter struct {
	url string
	h   piazza.Http
}

func NewHttpWriter(url string, apiKey string) (*HttpWriter, error) {
	w := &HttpWriter{url: url}
	w.h = piazza.Http{
		BaseUrl: url,
		ApiKey:  apiKey,
		//FlightCheck: &piazza.SimpleFlightCheck{},
	}

	return w, nil
}

func (w *HttpWriter) Write(mssg *Message, async bool) error {
	var _ Writer = (*HttpWriter)(nil)
	return aSyncLogic(w.writeWork, mssg, async)
}

func (w *HttpWriter) writeWork(mssg *Message) error {
	if jresp := w.h.PzPost("/syslog", mssg); jresp.IsError() {
		return jresp.ToError()
	}
	return nil
}

func (w *HttpWriter) Close() error {
	return nil
}

func (w *HttpWriter) Read(count int) ([]Message, error) {
	var _ Reader = (*HttpWriter)(nil)

	format := &piazza.JsonPagination{
		Page:    0,
		PerPage: count,
		SortBy:  "timeStamp",
		Order:   "desc",
	}
	params := &piazza.HttpQueryParams{}

	mssgs, _, err := w.GetMessages(format, params)
	return mssgs, err

}

// GetMessages is only implemented for HttpWriter as it is most likely the only
// Writer to be used for reading back data.
func (w *HttpWriter) GetMessages(
	format *piazza.JsonPagination,
	params *piazza.HttpQueryParams) ([]Message, int, error) {
	var _ Reader = (*HttpWriter)(nil)

	formatString := format.String()
	paramString := params.String()

	var ext string
	if formatString != "" && paramString != "" {
		ext = "?" + formatString + "&" + paramString
	} else if formatString == "" && paramString != "" {
		ext = "?" + paramString
	} else if formatString != "" && paramString == "" {
		ext = "?" + formatString
	} else if formatString == "" && paramString == "" {
		ext = ""
	} else {
		return nil, 0, errors.New("Internal error: failed to parse query params")
	}

	endpoint := "/syslog" + ext
	jresp := w.h.PzGet(endpoint)
	if jresp.IsError() {
		return nil, 0, jresp.ToError()
	}
	var mssgs []Message
	err := jresp.ExtractData(&mssgs)
	if err != nil {
		return nil, 0, err
	}

	return mssgs, jresp.Pagination.Count, nil
}

//---------------------------------------------------------------------

// SyslogdWriter implements a Writer that writes to the syslogd system service.
// This will almost certainly not work on Windows, but that is okay because Piazza
// does not support Windows.
type SyslogdWriter struct {
	writer *DaemonWriter
}

func (w *SyslogdWriter) initWriter() error {
	if w.writer != nil {
		return nil
	}

	tw, err := Dial(SyslogdNetwork, SyslogdRaddr)
	if err != nil {
		return err
	}

	w.writer = tw

	return nil
}

// Write writes the message to the OS's syslogd system.
func (w *SyslogdWriter) Write(mssg *Message, async bool) error {
	var _ Writer = (*SyslogdWriter)(nil)
	return aSyncLogic(w.writeWork, mssg, async)
}

func (w *SyslogdWriter) writeWork(mssg *Message) error {
	if err := w.initWriter(); err != nil {
		return err
	}
	s := mssg.String()
	return w.writer.Write(s)
}

// Close closes the underlying network connection.
func (w *SyslogdWriter) Close() error {
	if w.writer == nil {
		return nil
	}
	return w.writer.Close()
}

//---------------------------------------------------------------------

// ElasticWriter implements the Writer, writing to elasticsearch
type ElasticWriter struct {
	Esi elasticsearch.IIndex
	typ string
	id  string
}

func NewElasticWriter(esi elasticsearch.IIndex, typ string) *ElasticWriter {
	ew := &ElasticWriter{
		Esi: esi,
		typ: typ,
	}
	return ew
}

// Write writes the message to the elasticsearch index, type, id
func (w *ElasticWriter) Write(mssg *Message, async bool) error {
	var _ Writer = (*ElasticWriter)(nil)
	return aSyncLogic(w.writeWork, mssg, async)
}

func (w *ElasticWriter) writeWork(mssg *Message) error {
	if w == nil || w.Esi == nil || w.typ == "" {
		return fmt.Errorf("writer not set not set")
	}

	_, err := w.Esi.PostData(w.typ, w.id, mssg)
	return err
}

// SetType sets the type to write to
func (w *ElasticWriter) SetType(typ string) error {
	if w == nil {
		return fmt.Errorf("writer not set not set")
	}
	w.typ = typ
	return nil
}

func (w *ElasticWriter) CreateIndex() (bool, error) {
	if w == nil || w.Esi == nil {
		return false, fmt.Errorf("writer not set not set")
	}
	exists, err := w.Esi.IndexExists()
	if err != nil {
		return exists, err
	}
	if !exists {
		return exists, w.Esi.Create("")
	}
	return exists, nil
}

func (w *ElasticWriter) CreateType(mapping string) (bool, error) {
	if w == nil || w.Esi == nil || w.typ == "" {
		return false, fmt.Errorf("writer not set not set")
	}
	exists, err := w.Esi.TypeExists(w.typ)
	if err != nil {
		return exists, err
	}
	if exists {
		var currentMapping, newMapi interface{}
		if currentMapping, err = w.Esi.GetMapping(w.typ); err != nil {
			return exists, err
		}
		if newMapi, err = piazza.StructStringToInterface(mapping); err != nil {
			return exists, err
		}
		newMap := map[string]interface{}{w.typ: newMapi}
		if !reflect.DeepEqual(currentMapping, newMap) {
			return exists, errors.New("Elasticsearch contains an invalid mapping for type: " + w.typ)
		}
	} else {
		return exists, w.Esi.SetMapping(w.typ, piazza.JsonString(mapping))
	}
	return exists, nil
}

// SetID sets the id to write to
func (w *ElasticWriter) SetID(id string) error {
	if w == nil {
		return fmt.Errorf("writer not set not set")
	}
	w.id = id
	return nil
}

// Close does nothing but satisfy an interface.
func (w *ElasticWriter) Close() error {
	return nil
}

func (w *ElasticWriter) GetMessages(*piazza.JsonPagination, *piazza.HttpQueryParams) ([]Message, int, error) {
	return nil, 0, fmt.Errorf("ElasticWriter.GetMessages not supported")
}
func (w *ElasticWriter) Read(count int) ([]Message, error) {
	return nil, fmt.Errorf("ElasticWriter.Read not supported")
}

//-------------------------------

// NilWriter doesn't do anything
type NilWriter struct {
}

func (*NilWriter) Write(*Message, bool) error {
	var _ Writer = (*NilWriter)(nil)
	return nil
}

func (*NilWriter) writeWork(*Message) error {
	return nil
}

func (*NilWriter) Close() error {
	return nil
}

//---------------------------------------------------------------------

// MultiWriter will write to N different Writers at once. When doing a
// Write or a Close, we call the function on all of them. If any have failed,
// we return the error from the first one that failed.
type MultiWriter struct {
	writers []Writer
}

func NewMultiWriter(ws []Writer) *MultiWriter {
	mw := &MultiWriter{}
	mw.writers = make([]Writer, len(ws))
	copy(mw.writers, ws)
	return mw
}

func (mw *MultiWriter) Write(m *Message, async bool) (err error) {
	var _ Writer = (*HttpWriter)(nil)
	for _, w := range mw.writers {
		e := w.Write(m, async)
		if e != nil && err != nil {
			err = e
		}
	}
	return err
}

func (mw *MultiWriter) writeWork(m *Message) (err error) {
	return nil
}

func (mw *MultiWriter) Close() error {
	var err error

	for _, w := range mw.writers {
		e := w.Close()
		if e != nil && err != nil {
			err = e
		}
	}

	return err
}
