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

type mockPresenter struct {
	statusCode int
	headers    http.Header
}

func (m mockPresenter) PresentHTTP(r *http.Request) httphandler.Response {
	return httphandler.Response{
		StatusCode: m.statusCode,
		Headers:    m.headers,
		Body:       []byte(fmt.Sprintf("got request with method %s on path %s", r.Method, r.URL.Path)),
	}
}

// TestWriterSucceeds tests that the Writer http.Handler successfully
// writes the response that it recieves.
func TestWriterSucceeds(t *testing.T) {
	tests := []struct {
		testScenario   string
		presenter      mockPresenter
		request        *http.Request
		wantStatusCode int
		wantHeaders    http.Header
		wantBody       string
	}{
		{
			testScenario: "writing the response succeeds",
			presenter: mockPresenter{
				statusCode: 432,
				headers:    nil,
			},
			request:        httptest.NewRequest(http.MethodGet, "/hello-world", nil),
			wantStatusCode: 432,
			wantHeaders:    http.Header(map[string][]string{}),
			wantBody:       "got request with method GET on path /hello-world",
		},
		{
			testScenario: "writing another the response succeeds",
			presenter: mockPresenter{
				statusCode: 500,
				headers: http.Header(map[string][]string{
					"Authorization":   []string{"Basic: lkjasldfkj:laksjdf"},
					"Content-Type":    []string{"application/json"},
					"multiple-values": []string{"one", "two", "three"},
				}),
			},
			request:        httptest.NewRequest(http.MethodPost, "/hey/there", nil),
			wantStatusCode: 500,
			wantHeaders: http.Header(map[string][]string{
				"Authorization":   []string{"Basic: lkjasldfkj:laksjdf"},
				"Content-Type":    []string{"application/json"},
				"Multiple-Values": []string{"one", "two", "three"},
			}),
			wantBody: "got request with method POST on path /hey/there",
		},
	}
	for i, test := range tests {
		errorMsg := func(str string, args ...interface{}) {
			t.Helper()
			t.Errorf("Running test %d, where %s:\n"+str, append([]interface{}{i, test.testScenario}, args...)...)
		}
		w := httptest.NewRecorder()
		sut := httphandler.Writer{
			Presenter:     test.presenter,
			WriteFailedFn: nil,
		}

		sut.ServeHTTP(w, test.request)

		if got, want := w.Code, test.wantStatusCode; got != want {
			errorMsg("got status code %v, wanted %v", got, want)
		}
		if got, want := w.HeaderMap, test.wantHeaders; !reflect.DeepEqual(got, want) {
			errorMsg("got header mapping %#v, wanted %#v", got, want)
		}
		if got, want := w.Body.String(), test.wantBody; got != want {
			errorMsg("got body: %s, wanted: %s", got, want)
		}
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

// TestWriterFails tests that when writing the response fails we call
// a function on the Writer and pass it the error.
func TestWriterFails(t *testing.T) {
	var errPassedToFn error = nil
	errFn := func(e error) {
		errPassedToFn = e
	}
	sut := httphandler.Writer{
		Presenter:     mockPresenter{},
		WriteFailedFn: errFn,
	}

	sut.ServeHTTP(errResponseWriter{}, httptest.NewRequest("does-not-matter", "/does-not-matter", nil))

	if got, want := fmt.Sprintf("%v", errPassedToFn), "non-nil error occurred when writing"; got != want {
		t.Errorf("got error msg: %s, wanted error msg: %s", got, want)
	}
}

// TestDispatcher tests that the Dispatcher dispatches to the
// appropriate presenter based off the http method and if there is
// none then returns the expected response.
func TestDispatcher(t *testing.T) {
	tests := []struct {
		testScenario      string
		methodToPresenter map[string]httphandler.Presenter
		notFoundFn        func(*http.Request) httphandler.Response
		request           *http.Request
		wantResp          httphandler.Response
	}{
		{
			testScenario: "dispatches on a GET request",
			methodToPresenter: map[string]httphandler.Presenter{
				http.MethodGet: mockPresenter{statusCode: 100, headers: nil},
			},
			notFoundFn: nil,
			request:    httptest.NewRequest(http.MethodGet, "/hello-there", nil),
			wantResp: httphandler.Response{
				StatusCode: 100,
				Headers:    nil,
				Body:       []byte("got request with method GET on path /hello-there"),
			},
		},
		{
			testScenario: "dispatches on a POST request",
			methodToPresenter: map[string]httphandler.Presenter{
				http.MethodPost: mockPresenter{statusCode: 101, headers: nil},
			},
			notFoundFn: nil,
			request:    httptest.NewRequest(http.MethodPost, "/hello-there-buddy", nil),
			wantResp: httphandler.Response{
				StatusCode: 101,
				Headers:    nil,
				Body:       []byte("got request with method POST on path /hello-there-buddy"),
			},
		},
		{
			testScenario:      "the recieved method is not recognized so a function is called to generate the response",
			methodToPresenter: map[string]httphandler.Presenter{},
			notFoundFn: func(r *http.Request) httphandler.Response {
				return httphandler.Response{
					StatusCode: http.StatusMethodNotAllowed,
					Headers:    nil,
					Body:       []byte(fmt.Sprintf("the method %s is not allowed", r.Method)),
				}
			},
			request: httptest.NewRequest(http.MethodPost, "/hello-there-buddy", nil),
			wantResp: httphandler.Response{
				StatusCode: http.StatusMethodNotAllowed,
				Headers:    nil,
				Body:       []byte("the method POST is not allowed"),
			},
		},
	}
	for i, test := range tests {
		errorMsg := func(str string, args ...interface{}) {
			t.Helper()
			t.Errorf("Running test %d, where %s:\n"+str, append([]interface{}{i, test.testScenario}, args...)...)
		}
		sut := httphandler.Dispatcher{
			MethodToPresenter:  test.methodToPresenter,
			MethodNotSupported: httphandler.PresenterFunc(test.notFoundFn),
		}

		gotResp := sut.PresentHTTP(test.request)

		if got, want := gotResp.StatusCode, test.wantResp.StatusCode; got != want {
			errorMsg("got status code %v, wanted %v", got, want)
		}
		if got, want := gotResp.Headers, test.wantResp.Headers; !reflect.DeepEqual(got, want) {
			errorMsg("got header mapping %+v, wanted %+v", got, want)
		}
		if got, want := string(gotResp.Body), string(test.wantResp.Body); got != want {
			errorMsg("got body: %s, wanted: %s", got, want)
		}
	}
}

type mockErrPresenter struct {
	status int
	err    error
}

func (m mockErrPresenter) ErrPresentHTTP(r *http.Request) (httphandler.Response, error) {
	return httphandler.Response{
		StatusCode: m.status,
		Headers:    nil,
		Body:       []byte(fmt.Sprintf("got %s request on path %s", r.Method, r.URL.Path)),
	}, m.err
}

type errFnWasInvoked struct {
	wasInvoked bool
	gotErr     error
}

func (e *errFnWasInvoked) handleError(err error) {
	e.wasInvoked = true
	e.gotErr = err
}

// TestErrHandlerSucceeds tests that the ErrHandler Presenter will return the
// expected response whether or not an error occurred when generating
// the response.
func TestErrHandlerSucceds(t *testing.T) {
	tests := []struct {
		testScenario     string
		errPresenter     mockErrPresenter
		errHandler       errFnWasInvoked
		presenterIfErr   httphandler.Presenter
		request          *http.Request
		wantResp         httphandler.Response
		wantErrFnInvoked bool
		wantErrMsgPassed string
	}{
		{
			testScenario: "the ErrPresenter returns a response which gets returned",
			errPresenter: mockErrPresenter{
				status: 1,
				err:    nil,
			},
			errHandler:     errFnWasInvoked{},
			presenterIfErr: nil,
			request:        httptest.NewRequest(http.MethodDelete, "/cool/path", nil),
			wantResp: httphandler.Response{
				StatusCode: 1,
				Headers:    nil,
				Body:       []byte("got DELETE request on path /cool/path"),
			},
			wantErrFnInvoked: false,
			wantErrMsgPassed: "",
		},
		{
			testScenario: "the ErrPresenter returns an error, a function is called to handle the error, and the expected response is written",
			errPresenter: mockErrPresenter{
				status: 3,
				err:    errors.New("non-nil error"),
			},
			errHandler: errFnWasInvoked{},
			presenterIfErr: httphandler.PresenterFunc(func(r *http.Request) httphandler.Response {
				return httphandler.Response{
					StatusCode: 599,
					Headers:    nil,
					Body:       []byte(fmt.Sprintf("unexpected error on %s %s", r.Method, r.URL.Path)),
				}
			}),
			request: httptest.NewRequest(http.MethodPatch, "/really/cool/path", nil),
			wantResp: httphandler.Response{
				StatusCode: 599,
				Headers:    nil,
				Body:       []byte("unexpected error on PATCH /really/cool/path"),
			},
			wantErrFnInvoked: true,
			wantErrMsgPassed: "non-nil error",
		},
	}
	for i, test := range tests {
		errorMsg := func(str string, args ...interface{}) {
			t.Helper()
			t.Errorf("Running test %d, where %s:\n"+str, append([]interface{}{i, test.testScenario}, args...)...)
		}
		sut := httphandler.ErrHandler{
			ErrPresenter: test.errPresenter,
			OnErrFn:      test.errHandler.handleError,
			RespWhenErr:  test.presenterIfErr,
		}

		gotResp := sut.PresentHTTP(test.request)

		if got, want := gotResp.StatusCode, test.wantResp.StatusCode; got != want {
			errorMsg("got status code %v, wanted %v", got, want)
		}
		if got, want := gotResp.Headers, test.wantResp.Headers; !reflect.DeepEqual(got, want) {
			errorMsg("got header mapping %+v, wanted %+v", got, want)
		}
		if got, want := string(gotResp.Body), string(test.wantResp.Body); got != want {
			errorMsg("got body: %s, wanted: %s", got, want)
		}
		if got, want := test.errHandler.wasInvoked, test.wantErrFnInvoked; got != want {
			errorMsg("error fn being invoked was %v", got)
		}
		if test.wantErrFnInvoked {
			if got, want := fmt.Sprintf("%+v", test.errHandler.gotErr), test.wantErrMsgPassed; got != want {
				errorMsg("passed error msg was: %s, wanted: %s", got, want)
			}
		}
	}
}