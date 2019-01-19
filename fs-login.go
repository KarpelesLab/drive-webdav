package main

import (
	"context"
	"fmt"
	"os"

	"golang.org/x/net/webdav"
)

func NewDriveLoginFS(h *HttpServer, url string, name string) webdav.FileSystem {
	fs := webdav.NewMemFS()

	f, err := fs.OpenFile(context.Background(), "/"+name+".url", os.O_WRONLY|os.O_CREATE, 0755)
	if err != nil {
		panic(err)
	}
	fmt.Fprintf(f, "[InternetShortcut]\nURL=%s\n", url)
	f.Close()

	return fs
}
