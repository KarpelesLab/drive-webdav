package main

import (
	"log"
	"net"
	"net/http"

	"golang.org/x/net/webdav"
)

type HttpServer struct {
	webdav.Handler
	l *net.TCPListener
}

func NewHttpServer() (*HttpServer, error) {
	l, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 50500})
	res := &HttpServer{l: l}
	res.Handler.FileSystem = NewDriveLoginFS(res, "http://www.perdu.com", "Click here to Login")
	res.Handler.LockSystem = webdav.NewMemLS()
	res.Handler.Logger = func(r *http.Request, err error) {
		if err != nil {
			log.Printf("webdav: %s", err)
		}
	}
	return res, err
}

func (h *HttpServer) Serve() error {
	return http.Serve(h.l, h)
}

func (h *HttpServer) String() string {
	return h.l.Addr().String()
}

func (h *HttpServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// r.Method can be:
	if r.Method == "GET" {
		r.Method = "PROPFIND"
	}
	h.Handler.ServeHTTP(w, r)
}
