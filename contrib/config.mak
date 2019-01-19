PROJECT_NAME=drive-webdav
DIST_ARCHS=linux_amd64 windows_386 windows_amd64 darwin_amd64

ifeq ($(TARGET_GOOS),windows)
GOLDFLAGS+=-H=windowsgui
endif
