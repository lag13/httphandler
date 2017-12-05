package httphandler_test

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/lag13/httphandler"
)

// TestWriterSucceeds tests that the Writer http.Handler successfully
// writes the response that it receives.
func TestWriterSucceeds(t *testing.T) {
	tests := []struct {
		name           string
		presenter      httphandler.Presenter
		request        *http.Request
		wantStatusCode int
		wantHeader     http.Header
		wantBody       string
	}{
		{
			name: "a 0 status code defaults to 200",
			presenter: httphandler.PresenterFunc(func(r *http.Request) httphandler.Response {
				return httphandler.Response{
					Body: []byte(fmt.Sprintf("got request with method %s on path %s", r.Method, r.URL.Path)),
				}
			}),
			request:        httptest.NewRequest(http.MethodPatch, "/hello-world", nil),
			wantStatusCode: 200,
			wantHeader:     http.Header(map[string][]string{}),
			wantBody:       "got request with method PATCH on path /hello-world",
		},
		{
			name: "writing the response succeeds",
			presenter: httphandler.PresenterFunc(func(r *http.Request) httphandler.Response {
				return httphandler.Response{
					StatusCode: 432,
					Body:       []byte(fmt.Sprintf("got request with method %s on path %s", r.Method, r.URL.Path)),
				}
			}),
			request:        httptest.NewRequest(http.MethodGet, "/hello-world", nil),
			wantStatusCode: 432,
			wantHeader:     http.Header(map[string][]string{}),
			wantBody:       "got request with method GET on path /hello-world",
		},
		{
			name: "writing a response with headers succeeds",
			presenter: httphandler.PresenterFunc(func(r *http.Request) httphandler.Response {
				return httphandler.Response{
					StatusCode: 500,
					Header: http.Header{
						"Authorization":   []string{"Basic: lkjasldfkj:laksjdf"},
						"Content-Type":    []string{"application/json"},
						"multiple-values": []string{"one", "two", "three"},
					},
					Body: []byte(fmt.Sprintf("got request with method %s on path %s", r.Method, r.URL.Path)),
				}
			}),
			request:        httptest.NewRequest(http.MethodPost, "/hey/there", nil),
			wantStatusCode: 500,
			wantHeader: http.Header(map[string][]string{
				"Authorization":   {"Basic: lkjasldfkj:laksjdf"},
				"Content-Type":    {"application/json"},
				"Multiple-Values": {"one", "two", "three"},
			}),
			wantBody: "got request with method POST on path /hey/there",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			sut := httphandler.Writer{
				Presenter: test.presenter,
				HandleErr: nil,
			}

			sut.ServeHTTP(w, test.request)

			if got, want := w.Code, test.wantStatusCode; got != want {
				t.Errorf("got status code %v, wanted %v", got, want)
			}
			if got, want := w.HeaderMap, test.wantHeader; !reflect.DeepEqual(got, want) {
				t.Errorf("got header mapping %#v, wanted %#v", got, want)
			}
			if got, want := w.Body.String(), test.wantBody; got != want {
				t.Errorf("got body: %s, wanted: %s", got, want)
			}
		})
	}
}

// errResponseWriter implements http.ResponseWriter and returns an
// error when writing the response.
type errResponseWriter struct{}

func (e errResponseWriter) Header() http.Header {
	return nil
}

func (e errResponseWriter) Write([]byte) (int, error) {
	return 0, errors.New("non-nil error occurred when writing")
}

func (e errResponseWriter) WriteHeader(int) {
}

// fnToHandleErr is a struct with a method to handle an error and
// records state when that method is called. Some functions in this
// package call a function like that in their execution if something
// goes wrong so having this construct makes it easier to check
// whether that function was invoked or not.
type fnToHandleErr struct {
	wasInvoked bool
	gotReq     *http.Request
	gotErr     error
}

func (f *fnToHandleErr) handleError(r *http.Request, err error) {
	f.wasInvoked = true
	f.gotReq = r
	f.gotErr = err
}

// TestWriterFails tests that when writing the response fails we call
// a function on the Writer and pass it the request and error and if
// no function to handle the error was given then nothing is done.
func TestWriterFails(t *testing.T) {
	fnErrHandler := fnToHandleErr{}
	sut := httphandler.Writer{
		Presenter: httphandler.PresenterFunc(func(*http.Request) httphandler.Response { return httphandler.Response{} }),
		HandleErr: fnErrHandler.handleError,
	}
	req := httptest.NewRequest("does-not-matter", "/does-not-matter", nil)

	sut.ServeHTTP(errResponseWriter{}, req)

	if got, want := fnErrHandler.gotReq, req; got != want {
		t.Errorf("got req: %#v, wanted %#v", got, want)
	}
	if got, want := fmt.Sprintf("%v", fnErrHandler.gotErr), "non-nil error occurred when writing"; got != want {
		t.Errorf("got error msg: %s, wanted error msg: %s", got, want)
	}

	// Making sure this does not panic.
	sut.HandleErr = nil
	sut.ServeHTTP(errResponseWriter{}, req)
}

// TestDefaultResp tests that DefaultResp will produce the expected
// response from a presenter or a default response if that presenter
// returns a response with a status code of 0.
func TestDefaultResp(t *testing.T) {
	tests := []struct {
		name             string
		presenter        httphandler.Presenter
		defaultPresenter httphandler.Presenter
		request          *http.Request
		wantResp         httphandler.Response
	}{
		{
			name: "response comes from the presenter",
			presenter: httphandler.PresenterFunc(func(r *http.Request) httphandler.Response {
				return httphandler.Response{
					StatusCode: 101,
					Body:       []byte(fmt.Sprintf("got request with method %s on path %s", r.Method, r.URL.Path)),
				}
			}),
			defaultPresenter: nil,
			request:          httptest.NewRequest(http.MethodGet, "/whats-up-doc", nil),
			wantResp: httphandler.Response{
				StatusCode: 101,
				Header:     nil,
				Body:       []byte("got request with method GET on path /whats-up-doc"),
			},
		},
		{
			name: "presenter returns just body",
			presenter: httphandler.PresenterFunc(func(r *http.Request) httphandler.Response {
				return httphandler.Response{
					Body: []byte(fmt.Sprintf("got request with method %s on path %s", r.Method, r.URL.Path)),
				}
			}),
			defaultPresenter: httphandler.PresenterFunc(func(*http.Request) httphandler.Response {
				return httphandler.Response{
					StatusCode: 500,
					Body:       []byte("default response!"),
				}
			}),
			request: httptest.NewRequest(http.MethodGet, "/whats-up-doc", nil),
			wantResp: httphandler.Response{
				StatusCode: 0,
				Header:     nil,
				Body:       []byte("got request with method GET on path /whats-up-doc"),
			},
		},
		{
			name: "default response is returned",
			presenter: httphandler.PresenterFunc(func(r *http.Request) httphandler.Response {
				return httphandler.Response{}
			}),
			defaultPresenter: httphandler.PresenterFunc(func(*http.Request) httphandler.Response {
				return httphandler.Response{
					StatusCode: 500,
					Body:       []byte("default response!"),
				}
			}),
			request: httptest.NewRequest(http.MethodGet, "/whats-up-doc", nil),
			wantResp: httphandler.Response{
				StatusCode: 500,
				Header:     nil,
				Body:       []byte("default response!"),
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sut := httphandler.DefaultResp{
				Presenter:        test.presenter,
				DefaultPresenter: test.defaultPresenter,
			}

			gotResp := sut.PresentHTTP(test.request)

			if got, want := gotResp.StatusCode, test.wantResp.StatusCode; got != want {
				t.Errorf("got status code %v, wanted %v", got, want)
			}
			if got, want := gotResp.Header, test.wantResp.Header; !reflect.DeepEqual(got, want) {
				t.Errorf("got header mapping %+v, wanted %+v", got, want)
			}
			if got, want := string(gotResp.Body), string(test.wantResp.Body); got != want {
				t.Errorf("got body: %s, wanted: %s", got, want)
			}
		})
	}
}

// TestDispatcher tests that the Dispatcher dispatches to the
// appropriate presenter based off the http method and if there is
// none then returns the expected response.
func TestDispatcher(t *testing.T) {
	tests := []struct {
		name              string
		methodToPresenter map[string]httphandler.Presenter
		notFoundFn        func(*http.Request) httphandler.Response
		request           *http.Request
		wantResp          httphandler.Response
	}{
		{
			name: "dispatches on a GET request",
			methodToPresenter: map[string]httphandler.Presenter{
				http.MethodGet: httphandler.PresenterFunc(func(r *http.Request) httphandler.Response {
					return httphandler.Response{
						StatusCode: 100,
						Body:       []byte(fmt.Sprintf("got request with method %s on path %s", r.Method, r.URL.Path)),
					}
				}),
			},
			notFoundFn: nil,
			request:    httptest.NewRequest(http.MethodGet, "/hello-there", nil),
			wantResp: httphandler.Response{
				StatusCode: 100,
				Header:     nil,
				Body:       []byte("got request with method GET on path /hello-there"),
			},
		},
		{
			name: "dispatches on a POST request",
			methodToPresenter: map[string]httphandler.Presenter{
				http.MethodPost: httphandler.PresenterFunc(func(r *http.Request) httphandler.Response {
					return httphandler.Response{
						StatusCode: 101,
						Body:       []byte(fmt.Sprintf("got request with method %s on path %s", r.Method, r.URL.Path)),
					}
				}),
			},
			notFoundFn: nil,
			request:    httptest.NewRequest(http.MethodPost, "/hello-there-buddy", nil),
			wantResp: httphandler.Response{
				StatusCode: 101,
				Header:     nil,
				Body:       []byte("got request with method POST on path /hello-there-buddy"),
			},
		},
		{
			name:              "unrecognized http method",
			methodToPresenter: map[string]httphandler.Presenter{},
			notFoundFn: func(r *http.Request) httphandler.Response {
				return httphandler.Response{
					StatusCode: http.StatusMethodNotAllowed,
					Header:     nil,
					Body:       []byte(fmt.Sprintf("the method %s is not allowed", r.Method)),
				}
			},
			request: httptest.NewRequest(http.MethodPost, "/hello-there-buddy", nil),
			wantResp: httphandler.Response{
				StatusCode: http.StatusMethodNotAllowed,
				Header:     nil,
				Body:       []byte("the method POST is not allowed"),
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sut := httphandler.Dispatcher{
				MethodToPresenter:      test.methodToPresenter,
				MethodNotSupportedPres: httphandler.PresenterFunc(test.notFoundFn),
			}

			gotResp := sut.PresentHTTP(test.request)

			if got, want := gotResp.StatusCode, test.wantResp.StatusCode; got != want {
				t.Errorf("got status code %v, wanted %v", got, want)
			}
			if got, want := gotResp.Header, test.wantResp.Header; !reflect.DeepEqual(got, want) {
				t.Errorf("got header mapping %+v, wanted %+v", got, want)
			}
			if got, want := string(gotResp.Body), string(test.wantResp.Body); got != want {
				t.Errorf("got body: %s, wanted: %s", got, want)
			}
		})
	}
}

// TestErrHandler tests that the ErrHandler Presenter will return the
// expected response and handle an error if one occurrs.
func TestErrHandler(t *testing.T) {
	tests := []struct {
		name             string
		errPresenter     httphandler.ErrPresenter
		fnErrHandler     fnToHandleErr
		request          *http.Request
		wantResp         httphandler.Response
		wantErrFnInvoked bool
		wantErrMsgPassed string
	}{
		{
			name: "no error occur and got expected response",
			errPresenter: httphandler.ErrPresenterFunc(func(r *http.Request) (httphandler.Response, error) {
				return httphandler.Response{
					StatusCode: 1,
					Body:       []byte(fmt.Sprintf("got %s request on path %s", r.Method, r.URL.Path)),
				}, nil
			}),
			fnErrHandler: fnToHandleErr{},
			request:      httptest.NewRequest(http.MethodDelete, "/cool/path", nil),
			wantResp: httphandler.Response{
				StatusCode: 1,
				Header:     nil,
				Body:       []byte("got DELETE request on path /cool/path"),
			},
			wantErrFnInvoked: false,
			wantErrMsgPassed: "",
		},
		{
			name: "error occur and got expected response",
			errPresenter: httphandler.ErrPresenterFunc(func(r *http.Request) (httphandler.Response, error) {
				return httphandler.Response{
					Body: []byte(fmt.Sprintf("got %s request on path %s", r.Method, r.URL.Path)),
				}, errors.New("non-nil error")
			}),
			fnErrHandler: fnToHandleErr{},
			request:      httptest.NewRequest(http.MethodPatch, "/really/cool/path", nil),
			wantResp: httphandler.Response{
				StatusCode: 0,
				Header:     nil,
				Body:       []byte("got PATCH request on path /really/cool/path"),
			},
			wantErrFnInvoked: true,
			wantErrMsgPassed: "non-nil error",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sut := httphandler.ErrHandler{
				ErrPresenter: test.errPresenter,
				HandleErr:    test.fnErrHandler.handleError,
			}

			gotResp := sut.PresentHTTP(test.request)

			if got, want := gotResp.StatusCode, test.wantResp.StatusCode; got != want {
				t.Errorf("got status code %v, wanted %v", got, want)
			}
			if got, want := gotResp.Header, test.wantResp.Header; !reflect.DeepEqual(got, want) {
				t.Errorf("got header mapping %+v, wanted %+v", got, want)
			}
			if got, want := string(gotResp.Body), string(test.wantResp.Body); got != want {
				t.Errorf("got body: %s, wanted: %s", got, want)
			}
			if got, want := test.fnErrHandler.wasInvoked, test.wantErrFnInvoked; got != want {
				t.Errorf("error fn being invoked was %v", got)
			}
			if test.wantErrFnInvoked {
				if got, want := test.fnErrHandler.gotReq, test.request; got != want {
					t.Errorf("got req: %#v, wanted %#v", got, want)
				}
				if got, want := fmt.Sprintf("%+v", test.fnErrHandler.gotErr), test.wantErrMsgPassed; got != want {
					t.Errorf("passed error msg was: %s, wanted: %s", got, want)
				}
			}
		})
	}
}
