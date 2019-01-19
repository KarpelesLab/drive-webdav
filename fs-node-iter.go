package main

import (
	"io"
	"log"
	"os"
)

type fsNodeFolderIterator struct {
	children []os.FileInfo
	self     *fsNode
	pos      int
}

func (f *fsNodeFolderIterator) Close() error {
	return nil
}

func (f *fsNodeFolderIterator) Read(p []byte) (int, error) {
	return 0, os.ErrInvalid
}

func (f *fsNodeFolderIterator) Write(p []byte) (int, error) {
	return 0, os.ErrInvalid
}

func (f *fsNodeFolderIterator) Readdir(count int) ([]os.FileInfo, error) {
	log.Printf("Readdir(%d)", count)

	if count <= 0 {
		return f.children, nil
	}

	log.Printf("pos = %d count = %d", f.pos, len(f.children))

	var res []os.FileInfo
	if f.pos >= len(f.children) {
		return nil, io.EOF
	}
	for i := 0; i < count; i++ {
		if f.pos >= len(f.children) {
			break
		}
		res = append(res, f.children[f.pos])
		f.pos++
	}
	return res, nil
}

func (f *fsNodeFolderIterator) Seek(offset int64, whence int) (int64, error) {
	return 0, os.ErrInvalid
}

func (f *fsNodeFolderIterator) Stat() (os.FileInfo, error) {
	return f.self, nil
}
