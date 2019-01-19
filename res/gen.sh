#!/bin/sh
go get github.com/akavel/rsrc

mkdir -p windows_386
rsrc -ico icon.ico -arch 386 -o windows_386/rsrc_32.syso
echo "package windows_386" >windows_386/pkg.go

mkdir -p windows_amd64
rsrc -ico icon.ico -arch amd64 -o windows_amd64/rsrc_64.syso
echo "package windows_amd64" >windows_amd64/pkg.go
