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
func newHandler(mthdToErrPres map[string]httphandler.ErrPresenter) http.Handler {
	methodToPresenter := map[string]httphandler.Presenter{}
	for method, errPresenter := range mthdToErrPres {
		methodToPresenter[method] = httphandler.ErrHandler{
			ErrPresenter: errPresenter,
			HandleErr: func(r *http.Request, err error) {
				fmt.Printf("logging err on %s: %v\n", r.URL, err)
			},
		}
	}
	dispatcher := httphandler.Dispatcher{
		MethodToPresenter: methodToPresenter,
		MethodNotSupportedPres: httphandler.PresenterFunc(func(r *http.Request) httphandler.Response {
			return httphandler.Response{
				StatusCode: http.StatusMethodNotAllowed,
				Body:       []byte(fmt.Sprintf("method %s not allowed", r.Method)),
			}
		}),
	}
	defaultResp := httphandler.DefaultResp{
		Presenter: dispatcher,
		DefaultPresenter: httphandler.PresenterFunc(func(r *http.Request) httphandler.Response {
			return httphandler.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       []byte("unexpected error occurred"),
			}
		}),
	}
	return httphandler.Writer{Presenter: defaultResp}
}

type getData struct{}

func (g getData) ErrPresentHTTP(r *http.Request) (httphandler.Response, error) {
	return httphandler.Response{
		StatusCode: 200,
		Body:       []byte("here is some data"),
	}, nil
}

type createData struct{}

func (c createData) ErrPresentHTTP(r *http.Request) (httphandler.Response, error) {
	return httphandler.Response{}, errors.New("error when creating data")
}

type updateData struct{}

func (u updateData) ErrPresentHTTP(r *http.Request) (httphandler.Response, error) {
	return httphandler.Response{
		StatusCode: 400,
		Body:       []byte("some custom error response"),
	}, errors.New("error when updating data")
}

func Example() {
	getHandler := newHandler(map[string]httphandler.ErrPresenter{
		http.MethodGet: getData{},
	})
	createHandler := newHandler(map[string]httphandler.ErrPresenter{
		http.MethodPost: createData{},
	})
	updateHandler := newHandler(map[string]httphandler.ErrPresenter{
		http.MethodPut: updateData{},
	})

	w := httptest.NewRecorder()
	fmt.Println("getting data")
	getHandler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/hello", nil))
	fmt.Println("status code:", w.Code)
	fmt.Printf("body: %s\n", w.Body.String())

	w = httptest.NewRecorder()
	fmt.Println("getting data with the wrong http method")
	getHandler.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/hello", nil))
	fmt.Println("status code:", w.Code)
	fmt.Printf("body: %s\n", w.Body.String())

	w = httptest.NewRecorder()
	fmt.Println("creating data, error occurs, and a default response is returned")
	createHandler.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/hey", nil))
	fmt.Println("status code:", w.Code)
	fmt.Printf("body: %s\n", w.Body.String())

	w = httptest.NewRecorder()
	fmt.Println("updating data, error occurs, and custom response is returned")
	updateHandler.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/hey/1", nil))
	fmt.Println("status code:", w.Code)
	fmt.Printf("body: %s\n", w.Body.String())

	// Output: getting data
	// status code: 200
	// body: here is some data
	// getting data with the wrong http method
	// status code: 405
	// body: method POST not allowed
	// creating data, error occurs, and a default response is returned
	// logging err on /hey: error when creating data
	// status code: 500
	// body: unexpected error occurred
	// updating data, error occurs, and custom response is returned
	// logging err on /hey/1: error when updating data
	// status code: 400
	// body: some custom error response
}
