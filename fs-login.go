package main

import (
	"context"
	"os"

	"golang.org/x/net/webdav"
)

func NewDriveLoginFS(h *HttpServer, url string, name string) webdav.FileSystem {
	fs := webdav.NewMemFS()

	f, err := fs.OpenFile(context.Background(), "/"+name+".url", os.O_WRONLY|os.O_CREATE, 0755)
	if err != nil {
		panic(err)
	}
	f.Write([]byte(url))
	f.Close()

	return fs
}
