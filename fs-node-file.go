package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/AtOnline/drive-webdav/oauth2"
	"golang.org/x/net/webdav"
)

type fsNodeFile struct {
	self *fsNode
	flag int
	perm os.FileMode

	// specific to uploads
	parent    *fsNode
	upload    map[string]interface{}
	buf       bytes.Buffer
	committed int64

	pos int64

	resp *http.Response
	rpos int64 // pos in response
}

func (f *fsNodeFile) Close() error {
	f.pos = 0
	return f.finalizeUpload()
}

func (f *fsNodeFile) finalizeUpload() error {
	if f.buf.Len() > 0 {
		if f.committed == 0 {
			if f.upload == nil {
				up, err := f.self.overwrite()
				if err != nil {
					return err
				}
				f.upload = up
			}
			var fs *DriveFS
			if f.self == nil {
				fs = f.parent.fs
			} else {
				fs = f.self.fs
			}

			// can just complete this in a single PUT
			req, err := http.NewRequest("PUT", f.upload["PUT"].(string), bytes.NewReader(f.buf.Bytes()))
			if err != nil {
				return err
			}
			// perform post with default client
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return err
			}
			resp.Body.Close()

			// notify upload completion
			final, err := fs.c.Rest(f.upload["Complete"].(string), "POST", oauth2.RestParam{})
			if err != nil {
				return err
			}

			// complete
			f.buf.Reset()
			f.upload = nil

			// add child if new upload
			if f.self == nil {
				f.parent.load()
				f.self = f.parent.addChild(final.Data.(map[string]interface{}), "")
			}
			return nil
		}
		return webdav.ErrNotImplemented
	}
	return nil
}

func (f *fsNodeFile) Read(d []byte) (int, error) {
	if f.flag&os.O_RDONLY != os.O_RDONLY && f.flag&os.O_RDWR != os.O_RDWR {
		return 0, os.ErrInvalid
	}

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
	err := f.finalizeUpload()
	if err != nil {
		return nil, err
	}
	if f.self == nil {
		return nil, os.ErrNotExist
	}
	return f.self, nil
}

func (f *fsNodeFile) Write(d []byte) (int, error) {
	if f.pos != f.committed+int64(f.buf.Len()) {
		// can't write here
		return 0, os.ErrInvalid
	}
	n, err := f.buf.Write(d)
	if n > 0 {
		f.pos += int64(n)
	}
	return n, err
}
