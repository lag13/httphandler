package httphandler_test

import (
	"errors"
	"log"
	"net/http"

	"github.com/lag13/httphandler"
)

type myErrPresenter struct{}

func (m myErrPresenter) ErrPresentHTTP(r *http.Request) (httphandler.Response, error) {
	return httphandler.Response{}, errors.New("non-nil error")
}

func Example() {
	router := http.NewServeMux()
	firstPresenter := httphandler.PresenterFunc(func(r *http.Request) httphandler.Response {
		headers := http.Header(map[string][]string{})
		headers.Add("Content-Type", "application/json")
		headers.Add("Authorization", "Basic: alskjdflsa:lllllll")
		return httphandler.Response{
			StatusCode: http.StatusInternalServerError,
			Headers:    headers,
			Body:       []byte("hello there!\n"),
		}
	})
	firstWriter := httphandler.Writer{
		Presenter: firstPresenter,
		WriteFailedFn: func(r *http.Request, e error) {
			log.Fatalf("writing response to %s %s request failed with err: %v", r.Method, r.URL.Path, e)
		},
	}
	router.Handle("/first", firstWriter)
	secondPresenter := httphandler.PresenterFunc(func(r *http.Request) httphandler.Response {
		return httphandler.Response{
			StatusCode: http.StatusOK,
			Body:       []byte("nice day isn't it!\n"),
		}
	})
	secondDispatcher := httphandler.Dispatcher{
		MethodToPresenter: map[string]httphandler.Presenter{
			http.MethodGet: secondPresenter,
		},
		MethodNotSupportedPres: httphandler.PresenterFunc(func(r *http.Request) httphandler.Response {
			return httphandler.Response{Body: []byte("http method not supported")}
		}),
	}
	secondWriter := httphandler.Writer{
		Presenter: secondDispatcher,
		WriteFailedFn: func(r *http.Request, e error) {
			log.Fatalf("writing response to %s %s request failed with err: %v", r.Method, r.URL.Path, e)
		},
	}
	router.Handle("/second", secondWriter)
	thirdErrHandler := httphandler.ErrHandler{
		ErrPresenter: myErrPresenter{},
		OnErrFn: func(r *http.Request, e error) {
			log.Printf("unexpected error on %s %s", r.Method, r.URL.Path, e)
		},
	}
	thirdPresenter := httphandler.DefaultResp{
		Presenter: thirdErrHandler,
		DefaultPresenter: httphandler.PresenterFunc(func(*http.Request) httphandler.Response {
			return httphandler.Response{
				Body: []byte("an unexpected error occured"),
			}
		}),
	}
	thirdWriter := httphandler.Writer{
		Presenter: thirdPresenter,
		WriteFailedFn: func(r *http.Request, e error) {
			log.Fatalf("writing response to %s %s request failed with err: %v", r.Method, r.URL.Path, e)
		},
	}
	router.Handle("/third", thirdWriter)
	log.Fatal(http.ListenAndServe("localhost:8080", router))
}
