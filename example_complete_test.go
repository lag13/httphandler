package httphandler_test

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/lag13/httphandler"
)

// newHandler constructs a new handler. Using this function to
// construct http handlers for a given API means all those http
// handlers will: write the same http response if the method is not
// supported, write the same generic "unepxected error" http response
// if something goes wrong, and logging of errors happens in one place
// instead of having to be done by each http handler. These feel like
// nice simplifications.
func newHandler(methodToErrPresenter map[string]httphandler.ErrPresenter) http.Handler {
	methodToPresenter := map[string]httphandler.Presenter{}
	for method, errPresenter := range methodToErrPresenter {
		methodToPresenter[method] = httphandler.ErrHandler{
			ErrPresenter: errPresenter,
			HandleErr: func(r *http.Request, err error) {
				fmt.Printf("on %s %s endpoint error happened: %v\n", r.Method, r.URL.Path, err)
			},
		}
	}
	dispatcher := httphandler.Dispatcher{
		MethodToPresenter: methodToPresenter,
		MethodNotSupportedPres: httphandler.PresenterFunc(func(r *http.Request) httphandler.Response {
			return httphandler.Response{
				StatusCode: http.StatusMethodNotAllowed,
				Body:       []byte(fmt.Sprintf("the method %s is not allowed on this endpoint", r.Method)),
			}
		}),
	}
	defaultResp := httphandler.DefaultResp{
		Presenter: dispatcher,
		DefaultPresenter: httphandler.PresenterFunc(func(r *http.Request) httphandler.Response {
			return httphandler.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       []byte(fmt.Sprintf("unexpected error on the %s %s endpoint", r.Method, r.URL)),
			}
		}),
	}
	return httphandler.Writer{
		Presenter: defaultResp,
		HandleErr: func(r *http.Request, err error) {
			fmt.Printf("writing response to the %s %s request: %v\n", r.Method, r.URL, err)
		},
	}
}

type getSomeData struct{}

func (g getSomeData) ErrPresentHTTP(r *http.Request) (httphandler.Response, error) {
	return httphandler.Response{StatusCode: 200, Body: []byte("here is some data")}, nil
}

type createSomeDataAndError struct{}

func (c createSomeDataAndError) ErrPresentHTTP(r *http.Request) (httphandler.Response, error) {
	return httphandler.Response{}, errors.New("some error occurred when creating data")
}

type updateSomeDataAndError struct{}

func (u updateSomeDataAndError) ErrPresentHTTP(r *http.Request) (httphandler.Response, error) {
	return httphandler.Response{StatusCode: 400, Body: []byte("some custom error response")}, errors.New("some error occurred when updating data")
}

func Example() {
	getSomeDataHandler := newHandler(map[string]httphandler.ErrPresenter{
		http.MethodGet: getSomeData{},
	})
	createSomeDataHandler := newHandler(map[string]httphandler.ErrPresenter{
		http.MethodPost: createSomeDataAndError{},
	})
	updateSomeDataHandler := newHandler(map[string]httphandler.ErrPresenter{
		http.MethodPut: updateSomeDataAndError{},
	})

	w := httptest.NewRecorder()
	fmt.Println("getting some data")
	getSomeDataHandler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/hello", nil))
	fmt.Println("status code:", w.Code)
	fmt.Printf("body: %s\n", w.Body.String())

	w = httptest.NewRecorder()
	fmt.Println("getting some data with the wrong http method")
	getSomeDataHandler.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/hello", nil))
	fmt.Println("status code:", w.Code)
	fmt.Printf("body: %s\n", w.Body.String())

	w = httptest.NewRecorder()
	fmt.Println("an error occurs when creating some data and a default error response is returned")
	createSomeDataHandler.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/hello-world", nil))
	fmt.Println("status code:", w.Code)
	fmt.Printf("body: %s\n", w.Body.String())

	w = httptest.NewRecorder()
	fmt.Println("an error occurs when creating some data and a custom error response is returned")
	updateSomeDataHandler.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/hello-world/1", nil))
	fmt.Println("status code:", w.Code)
	fmt.Printf("body: %s\n", w.Body.String())
	// Output: getting some data
	// status code: 200
	// body: here is some data
	// getting some data with the wrong http method
	// status code: 405
	// body: the method POST is not allowed on this endpoint
	// an error occurs when creating some data and a default error response is returned
	// on POST /hello-world endpoint error happened: some error occurred when creating data
	// status code: 500
	// body: unexpected error on the POST /hello-world endpoint
	// an error occurs when creating some data and a custom error response is returned
	// on PUT /hello-world/1 endpoint error happened: some error occurred when updating data
	// status code: 400
	// body: some custom error response
}
