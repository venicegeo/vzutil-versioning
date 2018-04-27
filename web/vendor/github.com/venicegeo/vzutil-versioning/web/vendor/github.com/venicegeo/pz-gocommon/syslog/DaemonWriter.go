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

// This code is dervied from Go's syslog.go:
//   Copyright 2009 The Go Authors. All rights reserved.
//   Use of this source code is governed by a BSD-style
//   license that can be found in the LICENSE file.
// Go's syslog package doesn't provide enough flexibility for setting
// the format's fields, e.g. it hard-codes the tag (application), omits
// the hostname, etc.

package syslog

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
)

// A DaemonWriter is a connection to a syslog server.
type DaemonWriter struct {
	network string
	raddr   string

	mu   sync.Mutex // guards conn
	conn serverConn

	numWritten int // used only as a unit testing spy
}

type serverConn interface {
	writeString(msg, nl string) error
	close() error
}

type netConn struct {
	local bool
	conn  net.Conn
}

// Dial establishes a connection to a log daemon by connecting to
// address raddr on the specified network.  Each write to the returned
// writer sends a log message.
// If network is empty, Dial will connect to the local syslog server.
func Dial(network, raddr string) (*DaemonWriter, error) {
	w := &DaemonWriter{
		network: network,
		raddr:   raddr,
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	err := w.connect()
	if err != nil {
		return nil, err
	}
	return w, err
}

// connect makes a connection to the syslog server.
// It must be called with w.mu held.
func (w *DaemonWriter) connect() (err error) {
	if w.conn != nil {
		// ignore err from close, it makes sense to continue anyway
		_ = w.conn.close()
		w.conn = nil
	}

	if w.network == "" {
		w.conn, err = unixSyslog()
	} else {
		var c net.Conn
		c, err = net.Dial(w.network, w.raddr)
		if err == nil {
			w.conn = &netConn{conn: c}
		}
	}
	return
}

// Close closes a connection to the syslog daemon.
func (w *DaemonWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.conn != nil {
		err := w.conn.close()
		w.conn = nil
		return err
	}
	return nil
}

func (w *DaemonWriter) Write(msg string) (err error) {
	_, err = w.writeAndRetry(msg)
	return err
}

func (w *DaemonWriter) writeAndRetry(msg string) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.conn != nil {
		if n, err := w.write(msg); err == nil {
			return n, err
		}
	}
	if err := w.connect(); err != nil {
		return 0, err
	}
	return w.write(msg)
}

func (w *DaemonWriter) write(msg string) (int, error) {
	// ensure it ends in a \n
	nl := ""
	if !strings.HasSuffix(msg, "\n") {
		nl = "\n"
	}

	err := w.conn.writeString(msg, nl)
	if err != nil {
		return 0, err
	}

	w.numWritten++

	// Note: return the length of the input, not the number of
	// bytes printed by Fprintf, because this must behave like
	// an io.Writer.
	return len(msg), nil
}

func (n *netConn) writeString(msg, nl string) error {

	_, err := fmt.Fprintf(n.conn, "%s%s", msg, nl)
	return err
}

func (n *netConn) close() error {
	return n.conn.Close()
}

// unixSyslog opens a connection to the syslog daemon running on the
// local machine using a Unix domain socket.
func unixSyslog() (conn serverConn, err error) {
	logTypes := []string{"unixgram", "unix"}
	logPaths := []string{"/dev/log", "/var/run/syslog", "/var/run/log"}
	for _, network := range logTypes {
		for _, path := range logPaths {
			conn, err := net.Dial(network, path)
			if err != nil {
				continue
			} else {
				return &netConn{conn: conn, local: true}, nil
			}
		}
	}
	return nil, errors.New("Unix syslog delivery error")
}
