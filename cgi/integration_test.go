// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Tests a Go CGI program running under a Go CGI host process.
// Further, the two programs are the same binary, just checking
// their environment to figure out what mode to run in.

package cgi

import (
	"errors"
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	"github.com/ooni/oohttp"
	"github.com/ooni/oohttp/httptest"
)

type customWriterRecorder struct {
	w io.Writer
	*httptest.ResponseRecorder
}

func (r *customWriterRecorder) Write(p []byte) (n int, err error) {
	return r.w.Write(p)
}

type limitWriter struct {
	w io.Writer
	n int
}

func (w *limitWriter) Write(p []byte) (n int, err error) {
	if len(p) > w.n {
		p = p[:w.n]
	}
	if len(p) > 0 {
		n, err = w.w.Write(p)
		w.n -= n
	}
	if w.n == 0 {
		err = errors.New("past write limit")
	}
	return
}

// golang.org/issue/7198
func Test500WithNoHeaders(t *testing.T)     { want500Test(t, "/immediate-disconnect") }
func Test500WithNoContentType(t *testing.T) { want500Test(t, "/no-content-type") }
func Test500WithEmptyHeaders(t *testing.T)  { want500Test(t, "/empty-headers") }

func want500Test(t *testing.T, path string) {
	h := &Handler{
		Path: os.Args[0],
		Root: "/test.go",
		Args: []string{"-test.run=TestBeChildCGIProcess"},
	}
	expectedMap := map[string]string{
		"_body": "",
	}
	replay := runCgiTest(t, h, "GET "+path+" HTTP/1.0\nHost: example.com\n\n", expectedMap)
	if replay.Code != 500 {
		t.Errorf("Got code %d; want 500", replay.Code)
	}
}

type neverEnding byte

func (b neverEnding) Read(p []byte) (n int, err error) {
	for i := range p {
		p[i] = byte(b)
	}
	return len(p), nil
}

// Note: not actually a test.
func TestBeChildCGIProcess(t *testing.T) {
	if os.Getenv("REQUEST_METHOD") == "" {
		// Not in a CGI environment; skipping test.
		return
	}
	switch os.Getenv("REQUEST_URI") {
	case "/immediate-disconnect":
		os.Exit(0)
	case "/no-content-type":
		fmt.Printf("Content-Length: 6\n\nHello\n")
		os.Exit(0)
	case "/empty-headers":
		fmt.Printf("\nHello")
		os.Exit(0)
	}
	Serve(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.FormValue("nil-request-body") == "1" {
			fmt.Fprintf(rw, "nil-request-body=%v\n", req.Body == nil)
			return
		}
		rw.Header().Set("X-Test-Header", "X-Test-Value")
		req.ParseForm()
		if req.FormValue("no-body") == "1" {
			return
		}
		if eb, ok := req.Form["exact-body"]; ok {
			io.WriteString(rw, eb[0])
			return
		}
		if req.FormValue("write-forever") == "1" {
			io.Copy(rw, neverEnding('a'))
			for {
				time.Sleep(5 * time.Second) // hang forever, until killed
			}
		}
		fmt.Fprintf(rw, "test=Hello CGI-in-CGI\n")
		for k, vv := range req.Form {
			for _, v := range vv {
				fmt.Fprintf(rw, "param-%s=%s\n", k, v)
			}
		}
		for _, kv := range os.Environ() {
			fmt.Fprintf(rw, "env-%s\n", kv)
		}
	}))
	os.Exit(0)
}
