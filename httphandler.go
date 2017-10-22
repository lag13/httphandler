/*

Package httphandler provides components to help make http handlers.

Motivation

After writing and reading a number of http APIs written in Go I
started noticing duplicate code appearing across handlers (i.e types
that implement the http.Handler interface). Here is a sample handler
to illustrate what I mean:

	type SomeHandler struct{}

	func (s SomeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
		var statusCode int
		var body string
		var err error
		if r.Method == http.MethodGet {
			statusCode, body, err = s.handleGet(r)
		} else if r.Method == http.MethodPost {
			statusCode, body, err = s.handlePost(r)
		} else {
			statusCode = http.StatusMethodNotAllowed
		}
		if err != nil {
			log.Printf("an error occurred: %v", err)
			statusCode = http.StatusInternalServerError
			body = "unexpected error"
		}
		w.WriteHeader(statusCode)
		w.Write([]byte(body))
	}

	func (s SomeHandler) handleGet(r *http.Request) (int, string, error) {
		// handle the GET request
		return 0, "", nil
	}

	func (s SomeHandler) handlePost(r *http.Request) (int, string, error) {
		// handle the POST request
		return 0, "", nil
	}

The duplicate logic would always consist of some or all of the
following (all of which can be seen in the above code sample):

	1. Dispatch based off the request's method
	2. Log an error if one occurred
	3. Construct a response if an error occurs (when maybe the response should always be a generic "500 something went wrong" for a given API)
	4. Write the response (seems minor but you have to remember to add headers before calling WriteHeader() and similar "oddities")

This package was created to help address those 4 specific issues and
has a type to handle each case. These types can be composed together
to make a regular old http.Handler:

	1. Dispatcher
	2. ErrHandler
	3. DefaultResp
	4. Writer

All of the above utilities need not be used to define all handlers in
a given API (although Writer will probably be used if any of the
others are). If any specific handlers do not fit the mold that this
package introduces then by all means keep those handlers as they are
now.

Code Introduction

One driving idea behind this package is that handlers are simpler to
write and test if they return data instead of writing it. That gave
rise to the Presenter interface which is meant to be the functional
counterpart to http.Handler.

http.Handler:

	type Handler interface {
		ServeHTTP(ResponseWriter, *Request)
	}

Presenter:

	type Presenter interface {
		PresentHTTP(*http.Request) Response
	}

Since the response ultimately needs to get written this gave rise to
the "Writer" type who's sole purpose is to call
Presenter.PresentHTTP() and write the returned response. Writer is a
http.Handler and it serves as the "root" of any handlers created using
this package. With these two types as starting points this package
defines 3 instances of Presenter's which can be composed together with
specific handler logic to produce a complete http.Handler.

When defining an API specific handler using this package the handler
will probably implement the ErrPresenter interface which composes with
the ErrHandler Presenter. I urge you to read the source code of this
package (it is very short!) and see the examples to get a more
concrete understanding.

*/
package httphandler

import (
	"net/http"
)

// Response gets written in response to a request.
type Response struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
}

// Presenter will "present" (i.e show/return) the response that will
// be written. It is essentially the functional counterpart to the
// http.Handler interface.
type Presenter interface {
	PresentHTTP(*http.Request) Response
}

// Writer writes the response returned from a Presenter.
type Writer struct {
	Presenter Presenter
	HandleErr func(*http.Request, error)
}

// ServeHTTP writes the response received from a Presenter.
func (h Writer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	resp := h.Presenter.PresentHTTP(r)
	for header, values := range resp.Headers {
		for _, value := range values {
			w.Header().Add(header, value)
		}
	}
	// If Write() is called on a http.ResponseWriter before
	// WriteHeader() then a 200 status code is automatically
	// written. I stay consistent with that behavior by having
	// this if statement.
	if resp.StatusCode == 0 {
		resp.StatusCode = 200
	}
	w.WriteHeader(resp.StatusCode)
	// TODO: This error is never checked in the examples I've seen
	// (including the standard documentation) so maybe I should
	// default it to do nothing if no function is provided?
	if _, err := w.Write(resp.Body); err != nil {
		h.HandleErr(r, err)
	}
}

// DefaultResp is a Presenter which produces a response or a default
// response. The original purpose of this type was to use it to
// produce a generic "unexpected error occurred" response if something
// goes wrong, but perhaps there are other uses.
type DefaultResp struct {
	Presenter        Presenter
	DefaultPresenter Presenter
}

// PresentHTTP returns the response from a Presenter if the returned
// status code is non-zero otherwise it returns the response from a
// different Presenter.
func (d DefaultResp) PresentHTTP(r *http.Request) Response {
	resp := d.Presenter.PresentHTTP(r)
	if resp.StatusCode != 0 {
		return resp
	}
	return d.DefaultPresenter.PresentHTTP(r)
}

// Dispatcher is a Presenter which dispatches to another Presenter
// based on the http method.
type Dispatcher struct {
	MethodToPresenter      map[string]Presenter
	MethodNotSupportedPres Presenter
}

// PresentHTTP dispatches to another presenter based off the request's
// http method. If no matching presenter can be found a default
// response is returned.
func (d Dispatcher) PresentHTTP(r *http.Request) Response {
	p, ok := d.MethodToPresenter[r.Method]
	if !ok {
		return d.MethodNotSupportedPres.PresentHTTP(r)
	}
	return p.PresentHTTP(r)
}

// PresenterFunc allows the use of ordinary functions as Presenter's.
type PresenterFunc func(*http.Request) Response

// PresentHTTP calls f(r).
func (f PresenterFunc) PresentHTTP(r *http.Request) Response {
	return f(r)
}

// ErrPresenter is like a Presenter but returns (Response, error)
// instead of just Response. Most "real world" handlers (which might
// talk to a db or another service) will probably implement this
// interface or something similar to it.
type ErrPresenter interface {
	ErrPresentHTTP(*http.Request) (Response, error)
}

// ErrPresenterFunc allows the use of ordinary functions as
// ErrPresenter's.
type ErrPresenterFunc func(*http.Request) (Response, error)

// ErrPresentHTTP calls f(r).
func (f ErrPresenterFunc) ErrPresentHTTP(r *http.Request) (Response, error) {
	return f(r)
}

// ErrHandler is a Presenter that deals with errors.
type ErrHandler struct {
	ErrPresenter ErrPresenter
	HandleErr    func(*http.Request, error)
}

// PresentHTTP returns the response from an ErrPresenter and calls a
// function to handle the error if it is non-nil (this function could
// log the error for example).
func (e ErrHandler) PresentHTTP(r *http.Request) Response {
	resp, err := e.ErrPresenter.ErrPresentHTTP(r)
	if err != nil {
		e.HandleErr(r, err)
	}
	return resp
}
