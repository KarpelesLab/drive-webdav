// +build windows

package tray

import (
	"errors"
	"log"
	"runtime"
	"sync"
	"syscall"
	"unsafe"
)

// !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
// This is a customised version of https://github.com/xilp/systray/blob/master/tray_windows.go
// !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!

type Tray struct {
	id             uint32
	mhwnd          uintptr
	hwnd           uintptr
	tooltip        string
	onLeftClick    func()
	onRightClick   func()
	onDoubleClick  func()
	onTrayMenu     func() []MenuItem
	balloonClicked chan struct{}
	mut            sync.Mutex
}

func NewTray() (*Tray, error) {
	res := &Tray{
		onLeftClick:   func() {},
		onRightClick:  func() {},
		onDoubleClick: func() {},
		onTrayMenu: func() []MenuItem {
			return nil
		},
	}

	proc := make(chan error)

	// attempt to start event loop and wait for it to report success (chan closed) or error (error pushed)
	go res.eventLoop(proc)

	err, hasError := <-proc
	if hasError {
		return nil, err
	} else {
		return res, nil
	}
}

func (p *Tray) eventLoop(errCh chan error) {
	// This whole thing has to run on the same thread, as each thread has it's own UI queue?
	runtime.LockOSThread()

	err := p.init()
	if err != nil {
		// push error to channel, then close it
		errCh <- err
		close(errCh)
		return
	}

	// we reached that point, shouldn't be any error anymore
	close(errCh)

	hwnd := p.mhwnd
	var msg MSG
	for {
		rt, _, err := GetMessage.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		switch int(rt) {
		case 0:
			break
		case -1:
			log.Println("tray icon failed:", err.Error())
			break
		}

		is, _, _ := IsDialogMessage.Call(hwnd, uintptr(unsafe.Pointer(&msg)))
		if is == 0 {
			TranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
			DispatchMessage.Call(uintptr(unsafe.Pointer(&msg)))
		}
	}
}

// This needs to run on the same thread as the event loop.
func (ni *Tray) init() error {
	MainClassName := "MainForm"
	err := ni.registerWindow(MainClassName, ni.WinProc)
	if err != nil {
		return err
	}

	mhwnd, _, _ := CreateWindowEx.Call(
		WS_EX_CONTROLPARENT,
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(MainClassName))),
		0,
		WS_OVERLAPPEDWINDOW|WS_CLIPSIBLINGS,
		CW_USEDEFAULT,
		CW_USEDEFAULT,
		CW_USEDEFAULT,
		CW_USEDEFAULT,
		0,
		0,
		0,
		0)
	if mhwnd == 0 {
		return errors.New("create main win failed")
	}

	NotifyIconClassName := "NotifyIconForm"
	ni.registerWindow(NotifyIconClassName, ni.WinProc)

	hwnd, _, _ := CreateWindowEx.Call(
		0,
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(NotifyIconClassName))),
		0,
		0,
		0,
		0,
		0,
		0,
		uintptr(HWND_MESSAGE),
		0,
		0,
		0)
	if hwnd == 0 {
		return errors.New("create notify win failed")
	}

	nid := NOTIFYICONDATA{
		HWnd:             HWND(hwnd),
		UFlags:           NIF_MESSAGE | NIF_STATE,
		DwState:          NIS_HIDDEN,
		DwStateMask:      NIS_HIDDEN,
		UCallbackMessage: NotifyIconMessageId,
	}
	nid.CbSize = uint32(unsafe.Sizeof(nid))

	ret, _, _ := Shell_NotifyIcon.Call(NIM_ADD, uintptr(unsafe.Pointer(&nid)))
	if ret == 0 {
		return errors.New("shell notify create failed")
	}

	nid.UVersionOrTimeout = NOTIFYICON_VERSION

	ret, _, _ = Shell_NotifyIcon.Call(NIM_SETVERSION, uintptr(unsafe.Pointer(&nid)))
	if ret == 0 {
		ni.Stop()
		return errors.New("shell notify version failed")
	}

	ni.id = nid.UID
	ni.mhwnd = mhwnd
	ni.hwnd = hwnd

	icon, err := FindIcon()
	if err != nil {
		// fallback on yang
		icon, err = CreateIcon(yangIcon)
	}

	if err != nil {
		ni.Stop()
		return err
	}

	if err = ni.SetIcon(HICON(icon)); err != nil {
		ni.Stop()
		return err
	}

	return nil
}

func (p *Tray) Stop() {
	nid := NOTIFYICONDATA{
		UID:  p.id,
		HWnd: HWND(p.hwnd),
	}
	nid.CbSize = uint32(unsafe.Sizeof(nid))

	ret, _, err := Shell_NotifyIcon.Call(NIM_DELETE, uintptr(unsafe.Pointer(&nid)))
	if ret == 0 {
		log.Println("shell notify delete failed: ", err.Error())
	}
}

func (p *Tray) SetOnDoubleClick(fun func()) {
	if fun == nil {
		fun = func() {}
	}
	p.onDoubleClick = fun
}

func (p *Tray) SetOnLeftClick(fun func()) {
	if fun == nil {
		fun = func() {}
	}
	p.onLeftClick = fun
}

func (p *Tray) SetOnRightClick(fun func()) {
	if fun == nil {
		fun = func() {}
	}
	p.onRightClick = fun
}

func (p *Tray) SetMenuCreationCallback(fun func() []MenuItem) {
	if fun == nil {
		fun = func() []MenuItem { return nil }
	}
	p.onTrayMenu = fun
}

func (p *Tray) SetTooltip(tooltip string) error {
	nid := NOTIFYICONDATA{
		UID:  p.id,
		HWnd: HWND(p.hwnd),
	}
	nid.CbSize = uint32(unsafe.Sizeof(nid))

	nid.UFlags = NIF_TIP
	copy(nid.SzTip[:], syscall.StringToUTF16(tooltip))

	ret, _, err := Shell_NotifyIcon.Call(NIM_MODIFY, uintptr(unsafe.Pointer(&nid)))
	if ret == 0 {
		return err
	}
	return nil
}

func (p *Tray) SetVisible(visible bool) error {
	nid := NOTIFYICONDATA{
		UID:  p.id,
		HWnd: HWND(p.hwnd),
	}
	nid.CbSize = uint32(unsafe.Sizeof(nid))

	nid.UFlags = NIF_STATE
	nid.DwStateMask = NIS_HIDDEN
	if !visible {
		nid.DwState = NIS_HIDDEN
	}

	ret, _, err := Shell_NotifyIcon.Call(NIM_MODIFY, uintptr(unsafe.Pointer(&nid)))
	if ret == 0 {
		return err
	}
	return nil
}

func (p *Tray) SetIcon(hicon HICON) error {
	nid := NOTIFYICONDATA{
		UID:  p.id,
		HWnd: HWND(p.hwnd),
	}
	nid.CbSize = uint32(unsafe.Sizeof(nid))

	nid.UFlags = NIF_ICON
	if hicon == 0 {
		nid.HIcon = 0
	} else {
		nid.HIcon = hicon
	}

	ret, _, _ := Shell_NotifyIcon.Call(NIM_MODIFY, uintptr(unsafe.Pointer(&nid)))
	if ret == 0 {
		return errors.New("shell notify icon failed")
	}
	return nil
}

func (p *Tray) WinProc(hwnd HWND, msg uint32, wparam, lparam uintptr) uintptr {
	if msg == NotifyIconMessageId {
		switch lparam {
		case WM_LBUTTONDBLCLK:
			p.onDoubleClick()
		case WM_LBUTTONUP:
			p.onLeftClick()
		case WM_RBUTTONUP:
			p.onRightClick()
		case NIN_BALLOONUSERCLICK:
			fallthrough
		case NIN_BALLOONHIDE:
			fallthrough
		case NIN_BALLOONTIMEOUT:
			ch := p.balloonClicked
			if ch != nil {
				if lparam == NIN_BALLOONUSERCLICK {
					ch <- struct{}{}
				}
				close(ch)
			}
		}
	}
	result, _, _ := DefWindowProc.Call(uintptr(hwnd), uintptr(msg), wparam, lparam)
	return result
}

func (p *Tray) ShowNotification(title, message string, timeout int, onClick func()) error {
	nid := NOTIFYICONDATA{
		UID:  p.id,
		HWnd: HWND(p.hwnd),
	}
	nid.CbSize = uint32(unsafe.Sizeof(nid))

	nid.UFlags = NIF_INFO
	copy(nid.SzInfoTitle[:], syscall.StringToUTF16(title))
	copy(nid.SzInfo[:], syscall.StringToUTF16(message))
	nid.UVersionOrTimeout = uint32(timeout)

	if onClick != nil {
		p.mut.Lock()
		p.balloonClicked = make(chan struct{})
		// We hold the lock until the balloon has disappeared.
		go func() {
			select {
			case _, ok := <-p.balloonClicked:
				if ok {
					onClick()
				}
			}
			p.balloonClicked = nil
			p.mut.Unlock()
		}()
	}

	ret, _, _ := Shell_NotifyIcon.Call(NIM_MODIFY, uintptr(unsafe.Pointer(&nid)))
	if ret == 0 {
		return errors.New("shell notify notification failed")
	}
	return nil
}

func (p *Tray) registerWindow(name string, proc WindowProc) error {
	hinst, _, _ := GetModuleHandle.Call(0)
	if hinst == 0 {
		return errors.New("get module handle failed")
	}
	hicon, err := FindIcon()
	if err != nil {
		// fallback on yang
		hicon, err = CreateIcon(yangIcon)
	}

	if err != nil {
		return err
	}
	hcursor, _, _ := LoadCursor.Call(0, uintptr(IDC_ARROW))
	if hcursor == 0 {
		return errors.New("load cursor failed")
	}

	var wc WNDCLASSEX
	wc.CbSize = uint32(unsafe.Sizeof(wc))
	wc.LpfnWndProc = syscall.NewCallback(proc)
	wc.HInstance = HINSTANCE(hinst)
	wc.HIcon = HICON(hicon)
	wc.HCursor = HCURSOR(hcursor)
	wc.HbrBackground = COLOR_BTNFACE + 1
	wc.LpszClassName = syscall.StringToUTF16Ptr(name)

	atom, _, _ := RegisterClassEx.Call(uintptr(unsafe.Pointer(&wc)))
	if atom == 0 {
		return errors.New("register class failed")
	}
	return nil
}
