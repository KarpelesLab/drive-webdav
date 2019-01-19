package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/webdav"
)

type fsNode struct {
	fs   *DriveFS
	path string
	err  error // if any

	// info from API
	Id           string `json:"Drive_Item__"`
	Blob         string `json:"Blob__"`
	Type         string `json:"Type"`
	NodeName     string `json:"Name"`
	NodeSize     int64  `json:"Size"`
	LastModified time.Time

	// in case of directory, "children" is populated
	children map[string]*fsNode

	loadOnce sync.Once
}

func makeNode(v interface{}, path, name string, fs *DriveFS) *fsNode {
	r := &fsNode{
		fs:   fs,
		path: path,
	}

	vM := v.(map[string]interface{})

	r.Id = vM["Drive_Item__"].(string)
	r.Blob, _ = vM["Blob__"].(string) // directories won't have a blob
	r.Type = vM["Type"].(string)
	r.NodeName = name
	size, err := strconv.ParseInt(vM["Size"].(string), 0, 64)
	log.Printf("parse size = %d %s %s", size, vM["Size"], err)
	r.NodeSize = size
	r.LastModified = parseTime(vM["Last_Modified"])

	return r
}

func (n *fsNode) load() {
	// perform load
	n.loadOnce.Do(n.loadInternal)
}

func (n *fsNode) loadInternal() {
	switch n.path {
	case "/":
		// special case: list of drives
		n.Id = "Drive"
		n.Type = "folder"

		res, err := n.fs.c.Rest("Drive", "GET", RestParam{"results_per_page": "1000"})
		if err != nil {
			n.err = err
			return
		}

		// list of drives
		list := res.Data.([]interface{})
		n.children = make(map[string]*fsNode)

		// for each drive
		for _, info := range list {
			infoMap := info.(map[string]interface{})
			name := infoMap["Name"].(string)
			cnt := 1
			for {
				if _, found := n.children[name]; !found {
					break
				}
				// need to vary name
				cnt++
				name = fmt.Sprintf("%s (%d)", infoMap["Name"].(string), cnt)
				log.Printf("retry: %s", name)
			}
			node := makeNode(infoMap["Root"], "/"+name, name, n.fs)
			n.children[name] = node
		}
	default:
		log.Printf("unsupported access to node")
		n.err = webdav.ErrNotImplemented
	}
}

func (n *fsNode) get(path string) (*fsNode, error) {
	n.load()
	if path == "" || path == "/" {
		return n, nil
	}
	if n.Type != "folder" {
		// ... nope. can't browse inside a file
		return nil, os.ErrInvalid
	}
	path = strings.TrimLeft(path, "/")

	pos := strings.IndexByte(path, '/')
	if pos != -1 {
		// sub
		k := path[:pos]
		p, ok := n.children[k]
		if !ok {
			return nil, os.ErrNotExist
		}
		return p.get(path[pos+1:])
	}

	p, ok := n.children[path]
	if !ok {
		return nil, os.ErrNotExist
	}
	return p, nil
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
	return n.LastModified
}

func (n *fsNode) Name() string {
	return n.NodeName
}

func (n *fsNode) Size() int64 {
	log.Printf("get size = %d", n.NodeSize)
	return n.NodeSize
}

func (s *fsNode) Sys() interface{} {
	return nil
}

func (s *fsNode) ETag(ctx context.Context) (string, error) {
	if s.Blob == "" {
		return "", webdav.ErrNotImplemented
	}
	return "\"" + s.Blob + "\"", nil
}

func (n *fsNode) OpenFile(ctx context.Context, flag int, perm os.FileMode) (webdav.File, error) {
	if n.Type == "folder" {
		c := make([]os.FileInfo, len(n.children))
		pos := 0
		for _, sub := range n.children {
			c[pos] = sub
			pos++
		}
		return &fsNodeFolderIterator{self: n, children: c}, nil
	}
	return nil, webdav.ErrNotImplemented
}
