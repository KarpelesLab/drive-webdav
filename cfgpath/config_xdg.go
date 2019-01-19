// +build !windows,!darwin

package cfgpath

import (
	"os"
	"path/filepath"
)

// https://specifications.freedesktop.org/basedir-spec/basedir-spec-latest.html

var globalSettingFolder string
var cacheFolder string

func init() {
	if os.Getenv("XDG_CONFIG_HOME") != "" {
		globalSettingFolder = os.Getenv("XDG_CONFIG_HOME")
	} else {
		globalSettingFolder = filepath.Join(os.Getenv("HOME"), ".config")
	}
	if os.Getenv("XDG_CACHE_HOME") != "" {
		cacheFolder = os.Getenv("XDG_CACHE_HOME")
	} else {
		cacheFolder = filepath.Join(os.Getenv("HOME"), ".cache")
	}
}
