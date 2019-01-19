package main

import (
	"context"
	"os"
	"time"

	"golang.org/x/net/webdav"
)

type FileInfo struct {
	name    string
	size    int64
	modtime time.Time
	mode    os.FileMode
	etag    string
}

func (f *FileInfo) Name() string {
	return f.name
}

func (f *FileInfo) Size() int64 {
	return f.size
}

func (f *FileInfo) Mode() os.FileMode {
	return f.mode
}

func (f *FileInfo) ModTime() time.Time {
	return f.modtime
}

func (f *FileInfo) IsDir() bool {
	return f.mode.IsDir()
}

func (f *FileInfo) Sys() interface{} {
	return nil
}

func (f *FileInfo) ETag(ctx context.Context) (string, error) {
	if f.etag == "" {
		return "", webdav.ErrNotImplemented
	}
	return f.etag, nil
}
