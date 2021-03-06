package httphandler_test

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"

	"github.com/lag13/httphandler"
)

func ExampleWriter() {
	writer := httphandler.Writer{
		Presenter: httphandler.PresenterFunc(func(r *http.Request) httphandler.Response {
			return httphandler.Response{Body: []byte("hello world!")}
		}),
		HandleErr: func(r *http.Request, err error) {
			log.Printf("error on %s %s endpoint: %v", r.Method, r.URL, err)
		},
	}
	w := httptest.NewRecorder()
	writer.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/w", nil))
	fmt.Println("status code:", w.Code)
	fmt.Printf("body: %s\n", w.Body.String())

	// Output: status code: 200
	// body: hello world!
}

func ExampleDefaultResp() {
	defaultResp := httphandler.DefaultResp{
		Presenter: httphandler.PresenterFunc(func(r *http.Request) httphandler.Response {
			return httphandler.Response{}
		}),
		DefaultPresenter: httphandler.PresenterFunc(func(r *http.Request) httphandler.Response {
			return httphandler.Response{
				StatusCode: 500,
				Body:       []byte("generating default response"),
			}
		}),
	}
	resp := defaultResp.PresentHTTP(httptest.NewRequest(http.MethodPost, "/dr", nil))
	fmt.Println("status code:", resp.StatusCode)
	fmt.Printf("body: %s\n", resp.Body)

	// Output: status code: 500
	// body: generating default response
}

func ExampleDispatcher() {
	dispatcher := httphandler.Dispatcher{
		MethodToPresenter: map[string]httphandler.Presenter{
			http.MethodGet: httphandler.PresenterFunc(func(r *http.Request) httphandler.Response {
				return httphandler.Response{
					StatusCode: http.StatusOK,
					Body:       []byte("made it to a handler"),
				}
			}),
		},
		MethodNotSupportedPres: httphandler.PresenterFunc(func(r *http.Request) httphandler.Response {
			return httphandler.Response{
				StatusCode: http.StatusMethodNotAllowed,
				Body:       []byte("unsupported method"),
			}
		}),
	}
	resp := dispatcher.PresentHTTP(httptest.NewRequest(http.MethodGet, "/d", nil))
	fmt.Println("status code:", resp.StatusCode)
	fmt.Printf("body: %s\n", resp.Body)
	resp = dispatcher.PresentHTTP(httptest.NewRequest(http.MethodPost, "/d", nil))
	fmt.Println("status code:", resp.StatusCode)
	fmt.Printf("body: %s\n", resp.Body)

	// Output: status code: 200
	// body: made it to a handler
	// status code: 405
	// body: unsupported method
}

func ExampleErrHandler() {
	errHandler := httphandler.ErrHandler{
		ErrPresenter: httphandler.ErrPresenterFunc(func(r *http.Request) (httphandler.Response, error) {
			return httphandler.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       []byte("something went wrong"),
			}, errors.New("a bad error")
		}),
		HandleErr: func(r *http.Request, err error) {
			fmt.Printf("on %s %s endpoint got error: %v\n", r.Method, r.URL, err)
		},
	}
	resp := errHandler.PresentHTTP(httptest.NewRequest(http.MethodGet, "/eh", nil))
	fmt.Println("status code:", resp.StatusCode)
	fmt.Printf("body: %s\n", resp.Body)

	// Output: on GET /eh endpoint got error: a bad error
	// status code: 500
	// body: something went wrong
}
