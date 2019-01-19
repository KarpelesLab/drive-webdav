// +build windows

// those are types & calls from various windows system libs

package tray

import "syscall"

type WindowProc func(hwnd HWND, msg uint32, wparam, lparam uintptr) uintptr

type NOTIFYICONDATA struct {
	CbSize            uint32
	HWnd              HWND
	UID               uint32
	UFlags            uint32
	UCallbackMessage  uint32
	HIcon             HICON
	SzTip             [128]uint16
	DwState           uint32
	DwStateMask       uint32
	SzInfo            [256]uint16
	UVersionOrTimeout uint32
	SzInfoTitle       [64]uint16
	DwInfoFlags       uint32
	GuidItem          GUID
	HBalloonICon      HICON
}

type GUID struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

type WNDCLASSEX struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     HINSTANCE
	HIcon         HICON
	HCursor       HCURSOR
	HbrBackground HBRUSH
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       HICON
}

type MSG struct {
	HWnd    HWND
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      POINT
}

type POINT struct {
	X, Y int32
}

type (
	HANDLE    uintptr
	HINSTANCE HANDLE
	HCURSOR   HANDLE
	HICON     HANDLE
	HWND      HANDLE
	HGDIOBJ   HANDLE
	HBRUSH    HGDIOBJ
)

const (
	WM_LBUTTONUP     = 0x0202
	WM_LBUTTONDBLCLK = 0x0203
	WM_RBUTTONUP     = 0x0205
	WM_USER          = 0x0400

	WS_OVERLAPPEDWINDOW = 0X00000000 | 0X00C00000 | 0X00080000 | 0X00040000 | 0X00020000 | 0X00010000
	CW_USEDEFAULT       = 0x80000000

	NIM_ADD        = 0x00000000
	NIM_MODIFY     = 0x00000001
	NIM_DELETE     = 0x00000002
	NIM_SETVERSION = 0x00000004

	NIF_MESSAGE = 0x00000001
	NIF_ICON    = 0x00000002
	NIF_TIP     = 0x00000004
	NIF_STATE   = 0x00000008
	NIF_INFO    = 0x00000010

	NIN_BALLOONHIDE      = WM_USER + 3
	NIN_BALLOONTIMEOUT   = WM_USER + 4
	NIN_BALLOONUSERCLICK = WM_USER + 5

	NIS_HIDDEN = 0x00000001

	TPM_BOTTOMALIGN = 0x0020
	TPM_RETURNCMD   = 0x0100
	TPM_NONOTIFY    = 0x0080

	IDC_ARROW     = 32512
	COLOR_BTNFACE = 15

	WS_CLIPSIBLINGS     = 0X04000000
	WS_EX_CONTROLPARENT = 0X00010000

	HWND_MESSAGE       = ^HWND(2)
	NOTIFYICON_VERSION = 4

	WM_APP              = 32768
	NotifyIconMessageId = WM_APP + iota

	BFFM_INITIALIZED  = 1
	BFFM_SETSELECTION = WM_USER + 103

	// menu flags, see:
	// https://docs.microsoft.com/en-us/windows/desktop/api/winuser/nf-winuser-insertmenua
	MF_CHECKED      = 0x00000008
	MF_DISABLED     = 0x00000002
	MF_ENABLED      = 0x00000000
	MF_GRAYED       = 0x00000001
	MF_MENUBARBREAK = 0x00000020
	MF_MENUBREAK    = 0x00000040
	MF_OWNERDRAW    = 0x00000100
	MF_POPUP        = 0x00000010
	MF_SEPARATOR    = 0x00000800
	MF_STRING       = 0x00000000
	MF_UNCHECKED    = 0x00000000
)

var (
	kernel32        = syscall.MustLoadDLL("kernel32")
	GetModuleHandle = kernel32.MustFindProc("GetModuleHandleW")

	shell32          = syscall.MustLoadDLL("shell32.dll")
	Shell_NotifyIcon = shell32.MustFindProc("Shell_NotifyIconW")

	user32 = syscall.MustLoadDLL("user32.dll")

	GetMessage       = user32.MustFindProc("GetMessageW")
	IsDialogMessage  = user32.MustFindProc("IsDialogMessageW")
	TranslateMessage = user32.MustFindProc("TranslateMessage")
	DispatchMessage  = user32.MustFindProc("DispatchMessageW")

	DefWindowProc       = user32.MustFindProc("DefWindowProcW")
	RegisterClassEx     = user32.MustFindProc("RegisterClassExW")
	CreateWindowEx      = user32.MustFindProc("CreateWindowExW")
	SetForegroundWindow = user32.MustFindProc("SetForegroundWindow")
	GetCursorPos        = user32.MustFindProc("GetCursorPos")

	winCreateIcon = user32.MustFindProc("CreateIcon")
	LoadIcon      = user32.MustFindProc("LoadIconW")
	LoadCursor    = user32.MustFindProc("LoadCursorW")

	TrackPopupMenu  = user32.MustFindProc("TrackPopupMenu")
	CreatePopupMenu = user32.MustFindProc("CreatePopupMenu")
	AppendMenu      = user32.MustFindProc("AppendMenuW")

	SHParseDisplayName = shell32.MustFindProc("SHParseDisplayName")
)
