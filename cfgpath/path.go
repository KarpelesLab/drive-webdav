package cfgpath

import (
	"errors"
	"log"
	"os"
	"path/filepath"

	"github.com/TrisTech/goupd"
)

var initialPath string

func initPath() {
	getInitialPath()
	if goupd.PROJECT_NAME != "unconfigured" {
		// chdir to cache
		c := GetCacheDir()
		if err := EnsureDir(c); err != nil {
			log.Printf("[path] Failed to access cache directory: %s", err)
			return
		}
		log.Printf("[path] set cache dir: %s", c)
		os.Chdir(c)
	}
}

func getInitialPath() {
	initialPath, _ = os.Getwd()
	exe, err := os.Executable()
	if err != nil {
		log.Printf("[path] failed to get executable path: %s", err)
		return
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		log.Printf("[path] failed to parse executable path: %s", err)
		return
	}

	// get directory
	initialPath = filepath.Dir(exe)
}

func GetCacheDir() string {
	return filepath.Join(cacheFolder, goupd.PROJECT_NAME)
}

func GetConfigDir() string {
	return filepath.Join(globalSettingFolder, goupd.PROJECT_NAME)
}

func EnsureDir(c string) error {
	inf, err := os.Stat(c)
	if err != nil && os.IsNotExist(err) {
		err = os.MkdirAll(c, 0755)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else if err == nil && !inf.IsDir() {
		return errors.New("error: file exists at directory location")
	}
	return nil
}
