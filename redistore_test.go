package redistore

import (
	"bytes"
	"encoding/gob"
	"net/http"
	"testing"

	"github.com/gorilla/sessions"
)

// ----------------------------------------------------------------------------
// ResponseRecorder
// ----------------------------------------------------------------------------
// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// ResponseRecorder is an implementation of http.ResponseWriter that
// records its mutations for later inspection in tests.
type ResponseRecorder struct {
	Code      int           // the HTTP response code from WriteHeader
	HeaderMap http.Header   // the HTTP response headers
	Body      *bytes.Buffer // if non-nil, the bytes.Buffer to append written data to
	Flushed   bool
}

// NewRecorder returns an initialized ResponseRecorder.
func NewRecorder() *ResponseRecorder {
	return &ResponseRecorder{
		HeaderMap: make(http.Header),
		Body:      new(bytes.Buffer),
	}
}

// DefaultRemoteAddr is the default remote address to return in RemoteAddr if
// an explicit DefaultRemoteAddr isn't set on ResponseRecorder.
const DefaultRemoteAddr = "1.2.3.4"

// Header returns the response headers.
func (rw *ResponseRecorder) Header() http.Header {
	return rw.HeaderMap
}

// Write always succeeds and writes to rw.Body, if not nil.
func (rw *ResponseRecorder) Write(buf []byte) (int, error) {
	if rw.Body != nil {
		rw.Body.Write(buf)
	}
	if rw.Code == 0 {
		rw.Code = http.StatusOK
	}
	return len(buf), nil
}

// WriteHeader sets rw.Code.
func (rw *ResponseRecorder) WriteHeader(code int) {
	rw.Code = code
}

// Flush sets rw.Flushed to true.
func (rw *ResponseRecorder) Flush() {
	rw.Flushed = true
}

// ----------------------------------------------------------------------------

type FlashMessage struct {
	Type    int
	Message string
}

func TestNewRediStoreRoundOne(t *testing.T) {
	var req *http.Request
	var rsp *ResponseRecorder
	var hdr http.Header
	var err error
	var ok bool
	var cookies []string
	var session *sessions.Session
	var flashes []interface{}

	store, err := NewRediStore(10, "tcp", ":6379", "", []byte("secret-key"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.Ping()
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	req, err = http.NewRequest("GET", "http://localhost:8080/", nil)
	if err != nil {
		t.Fatal(err)
	}
	rsp = NewRecorder()
	// Get a session.
	if session, err = store.Get(req, "session-key"); err != nil {
		t.Fatalf("Error getting session: %v", err)
	}
	// Get a flash.
	flashes = session.Flashes()
	if len(flashes) != 0 {
		t.Errorf("Expected empty flashes; Got %v", flashes)
	}
	// Add some flashes.
	session.AddFlash("foo")
	session.AddFlash("bar")
	// Custom key.
	session.AddFlash("baz", "custom_key")
	session.AddFlash("baz2", "custom_key")
	session.AddFlash("baz3", "custom_key")
	session.AddFlash("baz4", "custom_key")
	session.AddFlash("baz5", "custom_key")
	session.AddFlash("baz6", "custom_key")
	// Save.
	if err = session.Save(req, rsp); err != nil {
		t.Fatalf("Error saving session: %v", err)
	}
	hdr = rsp.Header()
	cookies, ok = hdr["Set-Cookie"]
	if !ok || len(cookies) != 1 {
		t.Fatalf("No cookies. Header: %v", hdr)
	}
}

func TestNewRediStoreTwo(t *testing.T) {
	var req *http.Request
	var rsp *ResponseRecorder
	var err error
	var session *sessions.Session
	var flashes []interface{}

	store, err := NewRediStore(10, "tcp", ":6379", "", []byte("secret-key"))
	if err != nil {
		t.Fatal(err.Error())
	}
	_, err = store.Ping()
	if err != nil {
		t.Fatal(err.Error())
	}
	defer store.Close()
	rsp = NewRecorder()
	req, err = http.NewRequest("GET", "http://localhost:8080/", nil)
	if err != nil {
		t.Fatal(err.Error())
	}

	if session, err = store.Get(req, "session-key"); err != nil {
		t.Fatalf("Error getting session: %v", err)
	}

	session.AddFlash("foo")
	session.AddFlash("bar")
	// Custom key.
	session.AddFlash("baz", "custom_key")

	flashes = session.Flashes()
	if len(flashes) != 2 {
		t.Fatalf("Expected flashes; Got %v", flashes)
	}
	if flashes[0] != "foo" || flashes[1] != "bar" {
		t.Errorf("Expected foo,bar; Got %v", flashes)
	}
	flashes = session.Flashes()
	if len(flashes) != 0 {
		t.Errorf("Expected dumped flashes; Got %v", flashes)
	}
	// Custom key.
	flashes = session.Flashes("custom_key")
	if len(flashes) != 1 {
		t.Errorf("Expected flashes; Got %v", flashes)
	} else if flashes[0] != "baz" {
		t.Errorf("Expected baz; Got %v", flashes)
	}
	flashes = session.Flashes("custom_key")
	if len(flashes) != 0 {
		t.Errorf("Expected dumped flashes; Got %v", flashes)
	}

	// RediStore specific
	// Set MaxAge to -1 to mark for deletion.
	session.Options.MaxAge = -1
	// Save.
	if err = sessions.Save(req, rsp); err != nil {
		t.Fatalf("Error saving session: %v", err)
	}
}

func TestPingGoodPort(t *testing.T) {
	store, err := NewRediStore(10, "tcp", ":6379", "", []byte("secret-key"))
	if err != nil {
		t.Error(err.Error())
	}
	defer store.Close()
	ok, err := store.Ping()
	if err != nil {
		t.Error(err.Error())
	}
	if !ok {
		t.Error("Expected server to PONG")
	}
}

func TestPingBadPort(t *testing.T) {
	// we intentionally avoid checking errors here.
	store, _ := NewRediStore(10, "tcp", ":6378", "", []byte("secret-key"))
	defer store.Close()
	_, err := store.Ping()
	if err == nil {
		t.Error("Expected error")
	}
}

func ExampleRediStore() {
	// RedisStore
	store, err := NewRediStore(10, "tcp", ":6379", "", []byte("secret-key"))
	if err != nil {
		panic(err)
	}
	defer store.Close()
}

func init() {
	gob.Register(FlashMessage{})
}
