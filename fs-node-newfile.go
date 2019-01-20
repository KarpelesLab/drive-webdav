package main

import (
	"io"
	"log"
	"os"

	"golang.org/x/net/webdav"
)

type fsNodeNewFile struct {
	parent *fsNode
	name   string
	flag   int
	perm   os.FileMode
	url    string

	pos int64
}

func (f *fsNodeNewFile) Close() error {
	return nil
}

func (f *fsNodeNewFile) Read(d []byte) (int, error) {
	return 0, os.ErrInvalid
}

func (f *fsNodeNewFile) Readdir(count int) ([]os.FileInfo, error) {
	return nil, os.ErrInvalid
}

func (f *fsNodeNewFile) Seek(offset int64, whence int) (int64, error) {
	// cannot seek in a file being written
	switch whence {
	case io.SeekStart:
		return f.pos, os.ErrInvalid
	case io.SeekCurrent:
		if offset == 0 {
			return f.pos, nil
		}
		return f.pos, os.ErrInvalid
	case io.SeekEnd:
		return f.pos, os.ErrInvalid
	default:
		return f.pos, os.ErrInvalid
	}
}

func (f *fsNodeNewFile) Stat() (os.FileInfo, error) {
	return nil, os.ErrNotExist
}

func (f *fsNodeNewFile) Write(d []byte) (int, error) {
	log.Printf("Write len=%d pos=%d", len(d), f.pos)
	return 0, webdav.ErrNotImplemented
}
