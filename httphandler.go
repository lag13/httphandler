// Package httphandler contains some components to aid in making more
// functional http handlers.
package httphandler

import (
	"net/http"
)

// Response will ultimately get written to the wire in response to a
// request.
type Response struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
}

// Presenter will "present" (i.e show/return) the response that will
// be written. It is essentially the more functional counterpart to
// the http.Handler interface.
type Presenter interface {
	PresentHTTP(r *http.Request) Response
}

// Writer writes the response to the wire. Having this "side effect"
// logic exist in one place lets other handlers just return data which
// in my opinion makes them simpler.
type Writer struct {
	Presenter     Presenter
	WriteFailedFn func(*http.Request, error)
}

// ServeHTTP writes the response recieved from a Presenter.
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
	if _, err := w.Write(resp.Body); err != nil {
		h.WriteFailedFn(r, err)
	}
}

// DefaultResp is a Presenter which produces a response or a default
// response. The motivation for making this was to have a generic
// "error" response in case something goes wrong. That way, lower
// level handlers can just return an error and not worry about
// constructing a response. I'm sure there are other uses though.
type DefaultResp struct {
	Presenter        Presenter
	DefaultPresenter Presenter
}

// PresentHTTP returns the response from another Presenter or a
// default response if the first response has a status code of 0.
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

// ErrPresenter is like a Presenter but it returns (Response, error)
// instead of just Response. Most "real world" handlers (which might
// talk to a db or another service) will probably implement this
// interface.
type ErrPresenter interface {
	ErrPresentHTTP(r *http.Request) (Response, error)
}

// ErrHandler is a Presenter that deals with errors.
type ErrHandler struct {
	ErrPresenter ErrPresenter
	OnErrFn      func(*http.Request, error)
}

// PresentHTTP which will return the response from an ErrPresenter and
// call a function to handle the error (this function could log the
// error for example).
func (e ErrHandler) PresentHTTP(r *http.Request) Response {
	resp, err := e.ErrPresenter.ErrPresentHTTP(r)
	if err != nil {
		e.OnErrFn(r, err)
	}
	return resp
}
