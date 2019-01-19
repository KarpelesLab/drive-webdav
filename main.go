package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/AtOnline/drive-webdav/res"
	"github.com/AtOnline/drive-webdav/tray"
	"github.com/TrisTech/goupd"
)

var shutdownChannel = make(chan struct{})

func shutdown() {
	close(shutdownChannel)
}

func setupSignals() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGTERM)

	go func() {
		<-c
		shutdown()
	}()
}

func main() {
	setupSignals()
	goupd.AutoUpdate(false)

	t := tray.Init(shutdown)
	h, err := NewHttpServer()
	if err != nil {
		log.Printf("main: failed to create http server: %s", err)
	}

	log.Printf("main: listening on %s", h)
	log.Printf("login url: %s", h.LoginUrl())

	go h.Serve()

	<-shutdownChannel

	if t != nil {
		t.Stop()
	}

	h.Stop()
}
