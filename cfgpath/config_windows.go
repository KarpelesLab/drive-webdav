package cfgpath

import "os"

var globalSettingFolder = os.Getenv("APPDATA")
var cacheFolder = os.Getenv("LOCALAPPDATA")
