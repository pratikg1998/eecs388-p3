// This file has a lot of repetition, but we think that
// it makes it easier to linearly read what is happening
// in each test rather than passing control flow between functions.
// In an actual set of unit tests, you'd probably create
// helper functions.

package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	// The header used across all tests for
	// client-to-server communication.
	ctsHeaderKey   = "X-388-Request-Header"
	ctsHeaderValue = "request header value"
	// The header used across all tests for
	// server-to-client communication.
	stcHeaderKey   = "X-388-Response-Header"
	stcHeaderValue = "response header value"

	uri = "/test/uri"
)

func TestPassthroughRequest(t *testing.T) {
	type requestResult struct {
		request *http.Request
		body    string
	}

	body := "test body"
	r := httptest.NewRequest("TEST", uri, strings.NewReader(body))
	r.Header.Add(ctsHeaderKey, ctsHeaderValue)

	w := httptest.NewRecorder()

	requests := make(chan requestResult, 1)

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		requests <- requestResult{
			request: r,
			body:    string(b),
		}
		w.Header().Add(stcHeaderKey, stcHeaderValue)
		io.WriteString(w, "test response body")
	}))
	defer s.Close()

	PassthroughRequest(w, r, s.URL)

	var received requestResult
	// Wait up to 100 milliseconds for the response;
	// if we don't receive it by then, assume it's never coming.
	select {
	case received = <-requests:
	case <-time.After(100 * time.Millisecond):
		require.FailNow(t, "request not received by real server")
	}

	assert.Equal(t, "TEST", received.request.Method, "real server got wrong method")
	assert.Equal(t, uri, received.request.RequestURI, "real server got wrong URI")
	assert.Equal(t, ctsHeaderValue, received.request.Header.Get(ctsHeaderKey), "real server did not receive correct header value for key %s", ctsHeaderKey)
	assert.Equal(t, stcHeaderValue, w.Result().Header.Get(stcHeaderKey), "client did not receive correct header value in response for key %s", stcHeaderKey)
	assert.Equal(t, body, received.body, "real server got wrong request body")
	cl, _ := strconv.Atoi(w.Result().Header.Get("Content-Length"))
	assert.Equal(t, w.Body.Len(), cl, "client got response with declared Content-Length of %d bytes but actual body length of %d bytes", cl, w.Body.Len())
	assert.Equal(t, "test response body", w.Body.String(), "client got wrong response body")
}

func TestInterceptAndRelayNoChanges(t *testing.T) {
	type requestResult struct {
		request *http.Request
		body    url.Values
	}

	body := "real=test&loc=body"
	r := httptest.NewRequest("POST", uri, strings.NewReader(body))
	r.Header.Add(ctsHeaderKey, ctsHeaderValue)
	r.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	w := httptest.NewRecorder()

	requests := make(chan requestResult, 1)

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		v, _ := url.ParseQuery(string(b))
		requests <- requestResult{
			request: r,
			body:    v,
		}
		w.Header().Add(stcHeaderKey, stcHeaderValue)
		io.WriteString(w, "test response body")
	}))
	defer s.Close()

	InterceptAndRelayRequest(w, r, s.URL, "fake")

	var received requestResult
	select {
	case received = <-requests:
	case <-time.After(100 * time.Millisecond):
		require.FailNow(t, "request not received by real server")
	}

	assert.Equal(t, "POST", received.request.Method, "real server got wrong method")
	assert.Equal(t, uri, received.request.RequestURI, "real server got wrong URI")
	assert.Equal(t, ctsHeaderValue, received.request.Header.Get(ctsHeaderKey), "real server did not receive correct header value for key %s", ctsHeaderKey)
	assert.Equal(t, stcHeaderValue, w.Result().Header.Get(stcHeaderKey), "client did not receive correct header value in response for key %s", stcHeaderKey)
	ex, _ := url.ParseQuery(body)
	assert.EqualValues(t, ex, received.body, "real server got wrong request body")
	cl, _ := strconv.Atoi(w.Result().Header.Get("Content-Length"))
	assert.Equal(t, w.Body.Len(), cl, "client got response with declared Content-Length of %d bytes but actual body length of %d bytes", cl, w.Body.Len())
	assert.Equal(t, "test response body", w.Body.String(), "client got wrong response body")
}

func TestInterceptAndRelayChangeBoth(t *testing.T) {
	type requestResult struct {
		request *http.Request
		body    url.Values
	}

	body := "test=real&to=real"
	expectedAtServer := "test=real&to=not"
	expectedAtClient := "sabrina sent $1000 to real"
	r := httptest.NewRequest("POST", uri, strings.NewReader(body))
	r.Header.Add(ctsHeaderKey, ctsHeaderValue)
	r.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	w := httptest.NewRecorder()

	requests := make(chan requestResult, 1)

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		v, _ := url.ParseQuery(string(b))
		requests <- requestResult{
			request: r,
			body:    v,
		}
		w.Header().Add(stcHeaderKey, stcHeaderValue)
		io.WriteString(w, "sabrina sent $1000 to "+v.Get("to"))
	}))
	defer s.Close()

	InterceptAndRelayRequest(w, r, s.URL, "not")

	var received requestResult
	select {
	case received = <-requests:
	case <-time.After(100 * time.Millisecond):
		require.FailNow(t, "request not received by real server")
	}

	assert.Equal(t, "POST", received.request.Method, "real server got wrong method")
	assert.Equal(t, uri, received.request.RequestURI, "real server got wrong URI")
	assert.Equal(t, ctsHeaderValue, received.request.Header.Get(ctsHeaderKey), "real server did not receive correct header value for key %s", ctsHeaderKey)
	assert.Equal(t, stcHeaderValue, w.Result().Header.Get(stcHeaderKey), "client did not receive correct header value in response for key %s", stcHeaderKey)
	ex, _ := url.ParseQuery(expectedAtServer)
	assert.EqualValues(t, ex, received.body, "real server got wrong request body")
	cl, _ := strconv.Atoi(w.Result().Header.Get("Content-Length"))
	assert.Equal(t, w.Body.Len(), cl, "client got response with declared Content-Length of %d bytes but actual body length of %d bytes", cl, w.Body.Len())
	assert.Equal(t, expectedAtClient, w.Body.String(), "client got wrong response body")
}
