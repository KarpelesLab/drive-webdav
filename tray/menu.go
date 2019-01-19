// +build windows

package tray

import (
	"log"
	"syscall"
	"unsafe"
)

type MenuItem struct {
	Name     string
	Type     int
	State    int
	Children []MenuItem
	OnClick  func()
}

const TypeSubMenu = MF_POPUP

func (p *Tray) ShowMenu() {
	menuItems := p.onTrayMenu()
	if len(menuItems) == 0 {
		log.Println("No menu items, not showing menu")
		return
	}

	point := POINT{}
	if r0, _, err := GetCursorPos.Call(uintptr(unsafe.Pointer(&point))); r0 == 0 {
		log.Printf("failed to get mouse cursor position:", err.Error())
		return
	}

	callbacks := make(map[uintptr]func(), 0)

	menu := buildMenu(menuItems, callbacks)
	if menu == 0 {
		return
	}

	r0, _, err := SetForegroundWindow.Call(p.hwnd)
	if r0 == 0 {
		log.Printf("failed to bring window to foreground:", err.Error())
		return
	}

	r0, _, _ = TrackPopupMenu.Call(menu, TPM_BOTTOMALIGN|TPM_RETURNCMD|TPM_NONOTIFY, uintptr(point.X), uintptr(point.Y), 0, p.hwnd, 0)
	if r0 != 0 {
		if cb, ok := callbacks[r0]; ok && cb != nil {
			cb()
		}
	}
}

func buildMenu(items []MenuItem, callbacks map[uintptr]func()) uintptr {
	dropdown, _, err := CreatePopupMenu.Call()
	if dropdown == 0 {
		log.Println("failed to build a menu: ", err.Error())
		return 0
	}

	for _, item := range items {
		id := uintptr(0)
		if item.Type&TypeSubMenu == TypeSubMenu {
			id = buildMenu(item.Children, callbacks)
			if id == 0 {
				return 0
			}
		} else {
			for id = uintptr(1); id <= 9999999; id++ {
				if _, ok := callbacks[id]; !ok {
					break
				}
			}
		}
		callbacks[id] = item.OnClick
		r0, _, err := AppendMenu.Call(dropdown, uintptr(item.Type)|uintptr(item.State), uintptr(id), uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(item.Name))))
		if r0 == 0 {
			log.Println("failed to append menu item", err.Error())
			return 0
		}
	}
	return dropdown
}
