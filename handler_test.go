package delta

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/r7kamura/gospel"
	"github.com/r7kamura/router"
)

func setupServer(masterPort, shadowPort int) *Server {
	port := freePort()

	server := NewServer("0.0.0.0", port)

	server.AddMasterBackend("production", "0.0.0.0", masterPort)
	server.AddBackend("testing", "0.0.0.0", shadowPort)

	server.OnSelectBackend(func(req *http.Request) []string {
		if req.Method == "GET" {
			return []string{"production", "testing"}
		} else {
			return []string{"production"}
		}
	})

	server.OnMungeHeader(func(backend string, header *http.Header) {
		if backend == "testing" {
			header.Add("X-Delta-Sandbox", "1")
		}
	})
	return server
}

func launchBackend(backend string) (recorder *httptest.ResponseRecorder, port int) {
	port = freePort()
	router := router.NewRouter()
	recorder = httptest.NewRecorder()

	router.Get("/", http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
		fmt.Fprintf(recorder, "%s", backend)
	}))

	server := &http.Server{Addr: fmt.Sprintf(":%v", port), Handler: router}
	go server.ListenAndServe()

	return recorder, port
}

func freePort() int {
	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return 0
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func get(handler http.Handler, path string) *httptest.ResponseRecorder {
	return request(handler, "GET", path)
}

func request(handler http.Handler, method, path string) *httptest.ResponseRecorder {
	request, _ := http.NewRequest(method, path, nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

func TestHandler(t *testing.T) {
	productionResponse, productionPort := launchBackend("production")
	testingResponse, testingPort := launchBackend("testing")
	server := setupServer(productionPort, testingPort)
	handler := NewHandler(server)

	Describe(t, "ServeHTTP", func() {
		Context("when request to normal path", func() {
			get(handler, "/")

			It("should dispatch a request to production", func() {
				Expect(productionResponse.Body.String()).To(Equal, "production")
			})

			It("should dispatch a request to testing", func() {
				Expect(testingResponse.Body.String()).To(Equal, "testing")
			})

		})
	})
}
