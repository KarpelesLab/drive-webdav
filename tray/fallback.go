// +build linux darwin

package tray

type EmptyTray struct{}

func Init(func()) *EmptyTray {
	return &EmptyTray{}
}

func (e *EmptyTray) Stop() {}
