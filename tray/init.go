// +build windows

package tray

import (
	"log"

	"github.com/TrisTech/goupd"
)

var shutdownCallback func()

func Init(shutdownCb func()) *Tray {
	shutdownCallback = shutdownCb

	tray, err := NewTray()
	if err != nil {
		log.Printf("failed to init systray: %s", err)
		return nil
	}

	tray.SetVisible(true)
	tray.SetTooltip(goupd.PROJECT_NAME + " running (git#" + goupd.GIT_TAG + " released " + goupd.DATE_TAG + ")")

	tray.SetOnRightClick(tray.ShowMenu)

	tray.SetMenuCreationCallback(createMenu)

	return tray
}

func createMenu() []MenuItem {
	return []MenuItem{
		MenuItem{
			Name:    goupd.PROJECT_NAME + " running (git#" + goupd.GIT_TAG + " released " + goupd.DATE_TAG + ")",
			Type:    MF_STRING,
			State:   MF_DISABLED | MF_GRAYED,
			OnClick: func() {},
		},
		MenuItem{
			Name:    "Check for &Updates",
			Type:    MF_STRING,
			State:   MF_ENABLED,
			OnClick: func() { goupd.RunAutoUpdateCheck() },
		},
		MenuItem{
			Name:    "&Quit",
			Type:    MF_STRING,
			State:   MF_ENABLED,
			OnClick: shutdownCallback,
		},
	}
}
