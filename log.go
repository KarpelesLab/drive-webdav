package main

import (
	"io"
	"log"
	"os"

	"github.com/MagicalTux/ringbuf"
)

var logbuf *ringbuf.Writer

func init() {
	var err error

	logbuf, err = ringbuf.New(1024 * 1024)
	if err == nil {
		log.SetOutput(logbuf)
		go func() {
			r := logbuf.BlockingReader()
			defer r.Close()
			io.Copy(os.Stdout, r)
		}()
	} else {
		log.Printf("[log] Failed to setup logbuf: %s", err)
	}
}

func LogTarget() io.Writer {
	return logbuf
}

func LogDmesg(w io.Writer) (int64, error) {
	r := logbuf.Reader()
	defer r.Close()
	return io.Copy(w, r)
}
