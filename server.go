// Package delta is an HTTP shadow proxy server that sits between
// clients and your server(s) to enable "shadow requests".
package delta

import (
	"fmt"
	"log"
	"net"
	"net/http"
)

type Server struct {
	Host     string
	Port     int
	Backends map[string]*Backend

	waitForAllBackends bool

	onSelectBackendHandler   func(req *http.Request) []string
	onMungeHeaderHandler     func(backend string, header *http.Header)
	onBackendFinishedHandler func(map[string]*Response)
	onResponseHandler        func(responses ...*Response) *Response
}

func NewServer(host string, port int) *Server {
	server := &Server{
		Host: host,
		Port: port,
	}
	server.Backends = make(map[string]*Backend)

	// By default, all backends will be selected
	server.OnSelectBackend(func(req *http.Request) []string {
		backends := make([]string, 0)

		for key, _ := range server.Backends {
			backends = append(backends, key)
		}

		return backends
	})

	// by default return master backend
	server.OnResponse(func(responses ...*Response) *Response {
		for _, response := range responses {
			if response.Backend != nil && response.Backend.IsMaster {
				return response
			}
		}

		return nil
	})

	return server
}

func (server *Server) WaitForAllBackends(shouldWait bool) {
	server.waitForAllBackends = shouldWait
}

func (server *Server) AddMasterBackend(name, host string, port int) {
	server.Backends[name] = &Backend{
		IsMaster: true,
		Name:     name,
		Host:     host,
		Port:     port,
	}
}

func (server *Server) AddBackend(name, host string, port int) {
	server.Backends[name] = &Backend{
		IsMaster: false,
		Name:     name,
		Host:     host,
		Port:     port,
	}
}

func (server *Server) OnSelectBackend(handler func(req *http.Request) []string) {
	server.onSelectBackendHandler = handler
}

func (server *Server) OnMungeHeader(handler func(backend string, header *http.Header)) {
	server.onMungeHeaderHandler = handler
}

func (server *Server) OnBackendFinished(handler func(responses map[string]*Response)) {
	server.onBackendFinishedHandler = handler
}

func (server *Server) OnResponse(handler func(responses ...*Response) *Response) {
	server.onResponseHandler = handler
}

func (server *Server) Run() {
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", server.Host, server.Port))

	if err != nil {
		log.Fatal(err)
	}

	http.Handle("/", NewHandler(server))
	log.Fatal(http.Serve(listener, nil))
}
