// Package httphandler contains some components to aid in making more
// functional http handlers.
package httphandler

import (
	"net/http"
)

// Response will ultimately get written to the wire in response to a
// request. It will be returned from handlers created using this
// package.
type Response struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
}

// Presenter will "present" (i.e show/return) the response that should
// be written. It is essentially the more functional counterpart to
// the http.Handler interface.
type Presenter interface {
	PresentHTTP(r *http.Request) Response
}

// Writer writes the response to the wire. Having this "side effect"
// logic exist in one place lets other handlers just return data which
// in my opinion makes them simpler. The zero value of this struct can
// NOT be used.
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
	w.WriteHeader(resp.StatusCode)
	if _, err := w.Write(resp.Body); err != nil {
		h.WriteFailedFn(r, err)
	}
}

// Dispatcher is a Presenter which dispatches to a Presenter based on
// the http method. The zero value of this struct can NOT be used.
type Dispatcher struct {
	MethodToPresenter      map[string]Presenter
	MethodNotSupportedPres Presenter
}

// PresentHTTP dispatches to another presenter based off the request's
// http method.
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

// ErrHandler is a Presenter that deals with errors. The zero value of
// this struct can NOT be used.
type ErrHandler struct {
	ErrPresenter ErrPresenter
	OnErrFn      func(*http.Request, error)
	DefaultPres  Presenter
}

// PresentHTTP which will return the response from an ErrPresenter if
// that response's status code is non-zero otherwise it will generate
// some default response. If an error is also returned from the
// ErrPresenter then a function is called to handle that error
// (presumably log it).
func (e ErrHandler) PresentHTTP(r *http.Request) Response {
	resp, err := e.ErrPresenter.ErrPresentHTTP(r)
	if err != nil {
		e.OnErrFn(r, err)
	}
	if resp.StatusCode != 0 {
		return resp
	}
	return e.DefaultPres.PresentHTTP(r)
}
