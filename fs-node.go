package main

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/AtOnline/drive-webdav/oauth2"
	"golang.org/x/net/webdav"
)

type fsNode struct {
	fs  *DriveFS
	err error // if any

	// info from API
	name         string
	Id           string
	Blob         string
	Type         string
	url          string
	mime         string
	size         int64
	LastModified time.Time

	// in case of directory, "children" is populated
	children map[string]*fsNode

	loadOnce sync.Once
	driveId  string
	parent   *fsNode
	isRoot   bool
	refresh  time.Time
	refreshL sync.Mutex
}

func (r *fsNode) store(vM map[string]interface{}) {
	r.Id = vM["Drive_Item__"].(string)
	r.Type = vM["Type"].(string)
	if r.Type == "file" {
		r.Blob = vM["Blob__"].(string)      // directories/etc won't have a blob, ignore error
		r.url = vM["Download_Url"].(string) // only for files
		r.mime = vM["Mime"].(string)
	}
	r.name = vM["Name"].(string)
	r.LastModified = parseTime(vM["Last_Modified"])

	// size is returned as string
	size, _ := strconv.ParseInt(vM["Size"].(string), 0, 64)
	r.size = size
}

func makeNode(vM map[string]interface{}, name string, parent *fsNode) *fsNode {
	r := &fsNode{}
	r.store(vM)

	// set values
	r.name = name
	r.parent = parent
	r.driveId = parent.driveId
	r.fs = parent.fs

	return r
}

func (n *fsNode) load() {
	// perform load
	n.loadOnce.Do(n.loadInternal)
}

func (n *fsNode) reloadData() {
	n.load() // just in case

	n.refreshL.Lock()
	defer n.refreshL.Unlock()

	if time.Until(n.refresh) < 5*time.Second {
		// do not perform reload if did reload less than 5s ago
		return
	}

	n.loadInternal()
}

func (n *fsNode) addChild(infoMap map[string]interface{}, oname string) *fsNode {
	if oname == "" {
		oname = infoMap["Name"].(string)
	}
	name := oname
	cnt := 1
	for {
		if _, found := n.children[name]; !found {
			break
		}
		// need to vary name
		cnt++
		name = fmt.Sprintf("%s (%d)", oname, cnt)
		log.Printf("retry: %s", name)
	}
	node := makeNode(infoMap, name, n)
	if node != nil {
		n.children[name] = node
	}
	return node
}

func (n *fsNode) loadInternal() {
	if n.isRoot {
		n.initRoot()
		return
	}

	n.refresh = time.Now()

	switch n.Type {
	case "folder":
		// need to grab children
		res, err := n.fs.c.Rest("Drive/"+url.PathEscape(n.driveId)+"/Item", "GET", oauth2.RestParam{"Parent_Drive_Item__": n.Id, "results_per_page": "1000"})
		if err != nil {
			log.Printf("folder list failed: %s", err)
			n.err = err
			return
		}

		// list of drive items
		list := res.Data.([]interface{})
		n.children = make(map[string]*fsNode)

		log.Printf("found %d children", len(list))

		// for each drive
		for _, info := range list {
			n.addChild(info.(map[string]interface{}), "")
		}
	case "file":
		// nothing
	default:
		log.Printf("unsupported access to node")
		n.err = webdav.ErrNotImplemented
	}
}

func (n *fsNode) initRoot() {
	// special case: list of drives
	n.Id = "Drive"
	n.Type = "folder"

	res, err := n.fs.c.Rest("Drive", "GET", oauth2.RestParam{"results_per_page": "1000"})
	if err != nil {
		log.Printf("Failed to get drives list: %s", err)
		n.err = err
		return
	}

	// list of drives
	list := res.Data.([]interface{})
	n.children = make(map[string]*fsNode)

	// for each drive
	for _, info := range list {
		infoMap := info.(map[string]interface{})
		node := n.addChild(infoMap["Root"].(map[string]interface{}), infoMap["Name"].(string))
		if node != nil {
			node.driveId = infoMap["Drive__"].(string)
		}
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

func (n *fsNode) moveToTrash() error {
	// let's proceed
	if n.parent == nil || n.parent.isRoot {
		// invalid
		return os.ErrInvalid
	}

	_, err := n.fs.c.Rest("Drive/Item/"+url.PathEscape(n.Id), "DELETE", oauth2.RestParam{})
	if err != nil {
		return err
	}

	// remove from parent
	delete(n.parent.children, n.name)
	return nil
}

func (n *fsNode) Mode() os.FileMode {
	// TODO check rights, do not return write right if only read access
	switch n.Type {
	case "file":
		return 0755
	case "folder":
		return os.ModeDir | 0755
	case "special":
		return 0
	default:
		return 0
	}
}

func (n *fsNode) ModTime() time.Time {
	return n.LastModified
}

func (n *fsNode) Name() string {
	return n.name
}

func (n *fsNode) Size() int64 {
	return n.size
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

func (s *fsNode) ContentType(ctx context.Context) (string, error) {
	if s.mime == "" {
		return "", webdav.ErrNotImplemented
	}
	return s.mime, nil
}

func (n *fsNode) Mkdir(ctx context.Context, name string, perm os.FileMode) error {
	name = strings.TrimLeft(name, "/")
	name = strings.TrimRight(name, "/")

	pos := strings.IndexByte(name, '/')
	if pos != -1 {
		p, err := n.get(name[:pos])
		if err != nil {
			return err
		}
		return p.Mkdir(ctx, name[pos+1:], perm)
	}

	if n.isRoot {
		return os.ErrInvalid
	}

	// create dir
	res, err := n.fs.c.Rest("Drive/Item", "POST", oauth2.RestParam{"Name": name, "Parent_Drive_Item__": n.Id})
	if err != nil {
		// failed to create dir
		return err
	}

	// new dir created, reg it
	n.load()
	n.addChild(res.Data.(map[string]interface{}), "")
	return nil
}

func (n *fsNode) OpenFile(ctx context.Context, name string, flag int, perm os.FileMode) (webdav.File, error) {
	if name != "" {
		name = strings.TrimLeft(name, "/")
		pos := strings.IndexByte(name, '/')
		if pos != -1 {
			p, err := n.get(name[:pos])
			if err != nil {
				return nil, err
			}
			return p.OpenFile(ctx, name[pos+1:], flag, perm)
		}

		// check flags
		if flag&os.O_APPEND != 0 {
			log.Printf("no append")
			// cannot append with webdav
			return nil, webdav.ErrNotImplemented
		}

		// TODO handle file creation
		p, err := n.get(name)
		if err == nil {
			return p.OpenFile(ctx, "", flag, perm)
		}

		if flag&os.O_CREATE != 0 {
			// ok, let the user create a file
			res, err := oauth2.NewUpload(n.fs.c, "Drive/Item/"+url.PathEscape(n.Id)+":upload", oauth2.RestParam{"filename": name})
			if err != nil {
				return nil, err
			}
			return &fsNodeFile{parent: n, upload: res, flag: flag, perm: perm}, nil
		}
		return nil, err
	}

	switch n.Type {
	case "folder":
		log.Printf("return iterator")
		n.reloadData()
		c := make([]os.FileInfo, len(n.children))
		pos := 0
		for _, sub := range n.children {
			c[pos] = sub
			pos++
		}
		return &fsNodeFolderIterator{self: n, children: c}, nil
	case "file", "special":
		return &fsNodeFile{self: n, flag: flag, perm: perm}, nil
	default:
		return nil, os.ErrInvalid
	}
}

func (n *fsNode) overwrite() (*oauth2.Upload, error) {
	return oauth2.NewUpload(n.fs.c, "Drive/Item/"+url.PathEscape(n.Id)+":overwrite", nil)

}

func (n *fsNode) Rename(ctx context.Context, oldName, newName string) error {
	if oldName != "" {
		oldName = strings.TrimLeft(oldName, "/")
		pos := strings.IndexByte(oldName, '/')
		if pos != -1 {
			p, err := n.get(oldName[:pos])
			if err != nil {
				return err
			}
			return p.Rename(ctx, oldName[pos+1:], newName)
		}

		p, err := n.get(oldName)
		if err != nil {
			return err
		}
		return p.Rename(ctx, "", newName)
	}

	// get target dir
	tgt, err := n.fs.root.get(path.Dir(newName))
	if err != nil {
		return err
	}
	newName = path.Base(newName)
	if tgt == n.parent {
		// rename only
		if newName == n.name {
			// nothing?
			return nil
		}
		res, err := n.fs.c.Rest("Drive/Item/"+url.PathEscape(n.Id), "PATCH", oauth2.RestParam{"Name": newName})
		if err != nil {
			return err
		}
		// update (ugly, FIXME)
		delete(n.parent.children, n.name)
		n.name = res.Data.(map[string]interface{})["Name"].(string)
		n.parent.children[n.name] = n
		return nil
	}

	// use move API
	res, err := n.fs.c.Rest("Drive/Item/"+url.PathEscape(n.Id)+":moveTo", "POST", oauth2.RestParam{"target": tgt.Id, "rename": newName})
	if err != nil {
		return err
	}

	// update (ugly, FIXME)
	delete(n.parent.children, n.name)
	n.name = res.Data.(map[string]interface{})["Name"].(string)
	n.parent = tgt
	n.parent.children[n.name] = n
	return nil
}
