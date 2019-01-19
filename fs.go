package main

import (
	"context"
	"log"
	"os"
	"sync"

	"golang.org/x/net/webdav"
)

type DriveFS struct {
	c *OAuth2

	// cache path → node
	nodes  map[string]*fsNode
	nodesL sync.RWMutex
}

func NewDriveFS(c *OAuth2) *DriveFS {
	return &DriveFS{
		c:     c,
		nodes: make(map[string]*fsNode),
	}
}

func (fs *DriveFS) getPath(path string) (*fsNode, error) {
	fs.nodesL.RLock()
	d, ok := fs.nodes[path]
	fs.nodesL.RUnlock()

	if ok {
		d.load()
		return d, d.err
	}

	fs.nodesL.Lock()
	d, ok = fs.nodes[path]
	if ok {
		fs.nodesL.Unlock()
		d.load()
		return d, d.err
	}

	d = &fsNode{fs: fs, path: path}
	fs.nodes[path] = d
	fs.nodesL.Unlock()

	d.load()

	return d, d.err
}

func (fs *DriveFS) Mkdir(ctx context.Context, name string, perm os.FileMode) error {
	log.Printf("Mkdir(%s)", name)
	return webdav.ErrNotImplemented
}

func (fs *DriveFS) OpenFile(ctx context.Context, name string, flag int, perm os.FileMode) (webdav.File, error) {
	log.Printf("OpenFile(%s, %d, %s)", name, flag, perm)
	return nil, webdav.ErrNotImplemented
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
	return fs.getPath(name)
}
