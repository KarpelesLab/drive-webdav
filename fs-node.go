package main

import (
	"os"
	"sync"
	"time"
)

type srvDateTime struct {
	Unix int64 `json:"unix"`
}

type fsNode struct {
	fs   *DriveFS
	path string
	err  error // if any

	// info from API
	Id           string      `json:"Drive_Item__"`
	Blob         string      `json:"Blob__"`
	Type         string      `json:"Type"`
	NodeName     string      `json:"Name"`
	NodeSize     int64       `json:"Size"`
	LastModified srvDateTime `json:"Last_Modified"`

	loadOnce sync.Once
}

func (n *fsNode) load() {
	// perform load
	n.loadOnce.Do(n.loadInternal)
}

func (n *fsNode) loadInternal() {
	switch n.path {
	case "/":
		// special case: list of drives

	}
	// TODO
}

func (n *fsNode) IsDir() bool {
	return n.Type == "folder"
}

func (n *fsNode) Mode() os.FileMode {
	switch n.Type {
	case "file":
		return 0755
	case "folder":
		return os.ModeDir | 0755
	default:
		return 0
	}
}

func (n *fsNode) ModTime() time.Time {
	return time.Unix(n.LastModified.Unix, 0)
}

func (n *fsNode) Name() string {
	return n.NodeName
}

func (n *fsNode) Size() int64 {
	return n.NodeSize
}

func (s *fsNode) Sys() interface{} {
	return nil
}
