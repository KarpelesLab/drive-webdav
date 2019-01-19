package main

import (
	"log"
	"net"
	"net/http"
	"net/url"

	"golang.org/x/net/webdav"
)

type HttpServer struct {
	webdav.Handler
	l *net.TCPListener
}

func NewHttpServer() (*HttpServer, error) {
	l, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 50500})
	res := &HttpServer{l: l}
	res.Handler.FileSystem = NewDriveLoginFS(res, res.LoginUrl(), "Click here to Login")
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

func (h *HttpServer) LoginUrl() string {
	ret := "http://localhost:50500/_login"
	loginUrl := "https://hub.atonline.com/_special/rest/OAuth2:auth?response_type=code&client_id=oaap-k4ch3u-kibn-bovo-cb6t-uf463ufi&redirect_uri=" + url.QueryEscape(ret) + "&scope=profile+Drive"
	return loginUrl
}

func (h *HttpServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// r.Method can be:
	if r.Method == "GET" {
		if r.URL.Path == "/_login" {
			// TODO
			log.Printf("todo login")
		}
		r.Method = "PROPFIND"
	}
	h.Handler.ServeHTTP(w, r)
}
