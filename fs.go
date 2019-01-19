package main

import (
	"context"
	"log"
	"os"

	"golang.org/x/net/webdav"
)

type DriveFS struct {
	c *OAuth2

	// cache path → node
	root *fsNode
}

func NewDriveFS(c *OAuth2) *DriveFS {
	res := &DriveFS{
		c: c,
	}
	res.root = &fsNode{fs: res, isRoot: true}
	return res
}

func (fs *DriveFS) Mkdir(ctx context.Context, name string, perm os.FileMode) error {
	log.Printf("Mkdir(%s)", name)
	return webdav.ErrNotImplemented
}

func (fs *DriveFS) OpenFile(ctx context.Context, name string, flag int, perm os.FileMode) (webdav.File, error) {
	//log.Printf("OpenFile(%s, %d, %s)", name, flag, perm)
	p, err := fs.root.get(name)
	if err != nil {
		return nil, err
	}
	return p.OpenFile(ctx, flag, perm)
}

func (fs *DriveFS) RemoveAll(ctx context.Context, name string) error {
	log.Printf("RemoveAll(%s)", name)
	return webdav.ErrNotImplemented
}

func (fs *DriveFS) Rename(ctx context.Context, oldName, newName string) error {
	log.Printf("Rename(%s → %s", oldName, newName)
	return webdav.ErrNotImplemented
}

func (fs *DriveFS) Stat(ctx context.Context, name string) (os.FileInfo, error) {
	return fs.root.get(name)
}
