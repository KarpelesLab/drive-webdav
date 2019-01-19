package main

import (
	"fmt"
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

const (
	authEP      = "https://hub.atonline.com/_special/rest/OAuth2:auth"
	tokenEP     = "https://hub.atonline.com/_special/rest/OAuth2:token"
	clientId    = "oaap-k4ch3u-kibn-bovo-cb6t-uf463ufi"
	redirectUri = "http://localhost:50500/_login"
)

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
	loginUrl := authEP + "?response_type=code&client_id=" + url.QueryEscape(clientId) + "&redirect_uri=" + url.QueryEscape(redirectUri) + "&scope=profile+Drive"
	return loginUrl
}

func (h *HttpServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// r.Method can be:
	if r.Method == "GET" {
		if r.URL.Path == "/_login" {
			c, err := NewOAuth2(tokenEP, clientId, redirectUri, r.URL.Query().Get("code"))
			if err != nil {
				fmt.Fprintf(w, "Error authenticating: %s", err)
				return
			}
			fs := NewDriveFS(c)
			h.Handler.FileSystem = fs
			h.Handler.LockSystem = webdav.NewMemLS() // TODO
			fmt.Fprintf(w, "READY, you can now browse dav://%s", h)
			return
		}
		r.Method = "PROPFIND"
	}
	h.Handler.ServeHTTP(w, r)
}
