// +build windows

package tray

import (
	"errors"
	"unsafe"
)

type WinIcon struct {
	W, H        int
	Planes, Bpp byte
	AndT, XorT  []byte
}

func CreateIcon(i *WinIcon) (uintptr, error) {
	hicon, _, _ := winCreateIcon.Call(
		0,
		uintptr(i.W), uintptr(i.H),
		uintptr(i.Planes), uintptr(i.Bpp),
		uintptr(unsafe.Pointer(&i.AndT[0])),
		uintptr(unsafe.Pointer(&i.XorT[0])),
	)
	if hicon == 0 {
		return 0, errors.New("failed to initialize icon resource")
	}
	return hicon, nil
}

func FindIcon() (uintptr, error) {
	handle, _, _ := GetModuleHandle.Call(0)
	for i := uintptr(0); i <= 256; i++ {
		hicon, _, _ := LoadIcon.Call(handle, i)
		if hicon != 0 {
			return hicon, nil
		}
	}
	return 0, errors.New("failed to find icon")
}
