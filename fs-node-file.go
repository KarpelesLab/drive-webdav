package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"

	"golang.org/x/net/webdav"
)

type fsNodeFile struct {
	self *fsNode
	flag int
	perm os.FileMode

	pos int64

	resp *http.Response
	rpos int64 // pos in response
}

func (f *fsNodeFile) Close() error {
	return nil
}

func (f *fsNodeFile) Read(d []byte) (int, error) {
	// perform a read, intelligently (ha ha)
	if f.resp != nil {
		if f.pos > f.rpos && f.pos < (f.rpos+8*1024) {
			// we can read less than 8k of data to reach pos, that's probably faster than establishing a new http request
			drop := f.pos - f.rpos
			n, err := f.resp.Body.Read(make([]byte, drop))
			if n >= 0 {
				// with that, f.rpos should be == f.pos
				f.rpos += int64(n)
			}
			if err != nil {
				return 0, err
			}
		}
		// can we use this response?
		if f.rpos == f.pos {
			// yes.
			n, err := f.resp.Body.Read(d)
			if n > 0 {
				f.rpos += int64(n)
				f.pos += int64(n)
			}
			return n, err
		}

		// cannot use this response
		f.resp.Body.Close()
		f.resp = nil
	}

	if f.pos < 0 {
		// jsut in case, sanity check
		return 0, errors.New("negative seek not supported")
	}
	if f.pos >= f.self.size {
		// out of file
		return 0, io.EOF
	}

	if f.self.url == "" {
		// just do nothing since webdav doesn't like errors
		return len(d), nil
		//return 0, os.ErrPermission
	}

	req, err := http.NewRequest("GET", f.self.url, nil)
	if err != nil {
		return 0, err
	}

	if f.pos != 0 {
		// need to add range to request headers
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", f.pos))
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}

	f.resp = res
	f.rpos = f.pos

	// perform read
	n, err := f.resp.Body.Read(d)
	if n > 0 {
		f.rpos += int64(n)
		f.pos += int64(n)
	}
	return n, err
}

func (f *fsNodeFile) Readdir(count int) ([]os.FileInfo, error) {
	return nil, os.ErrInvalid
}

func (f *fsNodeFile) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		f.pos = offset
		return f.pos, nil
	case io.SeekCurrent:
		f.pos += offset
		return f.pos, nil
	case io.SeekEnd:
		f.pos = f.self.size + offset
		return f.pos, nil
	default:
		return f.pos, os.ErrInvalid
	}
}

func (f *fsNodeFile) Stat() (os.FileInfo, error) {
	return f.self, nil
}

func (f *fsNodeFile) Write(d []byte) (int, error) {
	return 0, webdav.ErrNotImplemented
}
