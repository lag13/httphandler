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
		WriteFailedFn: func(e error) {
			log.Fatal(e)
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
		MethodNotSupported: httphandler.PresenterFunc(func(r *http.Request) httphandler.Response {
			return httphandler.Response{Body: []byte("http method not supported")}
		}),
	}
	secondWriter := httphandler.Writer{
		Presenter: secondDispatcher,
		WriteFailedFn: func(e error) {
			log.Fatal(e)
		},
	}
	router.Handle("/second", secondWriter)
	thirdPresenterWhenErr := httphandler.PresenterFunc(func(r *http.Request) httphandler.Response {
		return httphandler.Response{
			Body: []byte("an unexpected error occured"),
		}
	})
	thirdPresenter := httphandler.ErrHandler{
		ErrPresenter: myErrPresenter{},
		OnErrFn: func(e error) {
			log.Print(e)
		},
		RespWhenErr: thirdPresenterWhenErr,
	}
	thirdWriter := httphandler.Writer{
		Presenter: thirdPresenter,
		WriteFailedFn: func(e error) {
			log.Fatal(e)
		},
	}
	router.Handle("/third", thirdWriter)
	log.Fatal(http.ListenAndServe("localhost:8080", router))
}
