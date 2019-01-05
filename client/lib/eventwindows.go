package eventwindows

import (
	"fmt"
	"syscall"

	"unsafe"

	//"github.com/golang/go/src/pkg/unsafe"

	"golang.org/x/sys/windows"
)

var (
	moduser32              = windows.NewLazyDLL("user32.dll")
	procPostThreadMessage  = moduser32.NewProc("PostThreadMessageA")
	modkernel32            = windows.NewLazyDLL("kernel32.dll")
	procGetCurrentThreadId = modkernel32.NewProc("GetCurrentThreadId")

	user32               = windows.NewLazySystemDLL("user32.dll")
	procSetWindowsHookEx = user32.NewProc("SetWindowsHookExW")
	procSetWinEventHook  = user32.NewProc("SetWinEventHook")

	procLowLevelKeyboard    = user32.NewProc("LowLevelKeyboardProc")
	procCallNextHookEx      = user32.NewProc("CallNextHookEx")
	procUnhookWindowsHookEx = user32.NewProc("UnhookWindowsHookEx")

	procGetMessage  = user32.NewProc("GetMessageW")
	procPeekMessage = user32.NewProc("PeekMessage")

	procQuitMessage      = user32.NewProc("PostQuitMessage")
	procTranslateMessage = user32.NewProc("TranslateMessage")
	procDispatchMessage  = user32.NewProc("DispatchMessageW")
	keyboardHook         HHOOK
)

type (
	HANDLE        uintptr
	HINSTANCE     HANDLE
	HHOOK         HANDLE
	HMODULE       HANDLE
	HWINEVENTHOOK HANDLE
	DWORD         uint32
	INT           int
	WPARAM        uintptr
	LPARAM        uintptr
	LRESULT       uintptr
	HWND          HANDLE
	UINT          uint32
	BOOL          int32
	ULONG_PTR     uintptr
	LONG          int32
	LPWSTR        *WCHAR
	WCHAR         uint16
)

const (
	WH_KEYBOARD_LL = 13
	WH_KEYBOARD    = 2
	WM_KEYDOWN     = 256
	WM_SYSKEYDOWN  = 260
	WM_KEYUP       = 257
	WM_SYSKEYUP    = 261
	WM_KEYFIRST    = 256
	WM_KEYLAST     = 264
	PM_NOREMOVE    = 0x000
	PM_REMOVE      = 0x001
	PM_NOYIELD     = 0x002
	WM_LBUTTONDOWN = 513
	WM_RBUTTONDOWN = 516
	NULL           = 0
)
const (
	HC_ACTION = 0
)

const (
	VK_ABNT_C1        = 0xC1
	VK_ABNT_C2        = 0xC2
	VK_ADD            = 0x6B
	VK_ATTN           = 0xF6
	VK_BACK           = 0x08
	VK_CANCEL         = 0x03
	VK_CLEAR          = 0x0C
	VK_CRSEL          = 0xF7
	VK_DECIMAL        = 0x6E
	VK_DIVIDE         = 0x6F
	VK_EREOF          = 0xF9
	VK_ESCAPE         = 0x1B
	VK_EXECUTE        = 0x2B
	VK_EXSEL          = 0xF8
	VK_ICO_CLEAR      = 0xE6
	VK_ICO_HELP       = 0xE3
	VK_KEY_0          = 0x30
	VK_KEY_1          = 0x31
	VK_KEY_2          = 0x32
	VK_KEY_3          = 0x33
	VK_KEY_4          = 0x34
	VK_KEY_5          = 0x35
	VK_KEY_6          = 0x36
	VK_KEY_7          = 0x37
	VK_KEY_8          = 0x38
	VK_KEY_9          = 0x39
	VK_KEY_A          = 0x41
	VK_KEY_B          = 0x42
	VK_KEY_C          = 0x43
	VK_KEY_D          = 0x44
	VK_KEY_E          = 0x45
	VK_KEY_F          = 0x46
	VK_KEY_G          = 0x47
	VK_KEY_H          = 0x48
	VK_KEY_I          = 0x49
	VK_KEY_J          = 0x4A
	VK_KEY_K          = 0x4B
	VK_KEY_L          = 0x4C
	VK_KEY_M          = 0x4D
	VK_KEY_N          = 0x4E
	VK_KEY_O          = 0x4F
	VK_KEY_P          = 0x50
	VK_KEY_Q          = 0x51
	VK_KEY_R          = 0x52
	VK_KEY_S          = 0x53
	VK_KEY_T          = 0x54
	VK_KEY_U          = 0x55
	VK_KEY_V          = 0x56
	VK_KEY_W          = 0x57
	VK_KEY_X          = 0x58
	VK_KEY_Y          = 0x59
	VK_KEY_Z          = 0x5A
	VK_MULTIPLY       = 0x6A
	VK_NONAME         = 0xFC
	VK_NUMPAD0        = 0x60
	VK_NUMPAD1        = 0x61
	VK_NUMPAD2        = 0x62
	VK_NUMPAD3        = 0x63
	VK_NUMPAD4        = 0x64
	VK_NUMPAD5        = 0x65
	VK_NUMPAD6        = 0x66
	VK_NUMPAD7        = 0x67
	VK_NUMPAD8        = 0x68
	VK_NUMPAD9        = 0x69
	VK_OEM_1          = 0xBA
	VK_OEM_102        = 0xE2
	VK_OEM_2          = 0xBF
	VK_OEM_3          = 0xC0
	VK_OEM_4          = 0xDB
	VK_OEM_5          = 0xDC
	VK_OEM_6          = 0xDD
	VK_OEM_7          = 0xDE
	VK_OEM_8          = 0xDF
	VK_OEM_ATTN       = 0xF0
	VK_OEM_AUTO       = 0xF3
	VK_OEM_AX         = 0xE1
	VK_OEM_BACKTAB    = 0xF5
	VK_OEM_CLEAR      = 0xFE
	VK_OEM_COMMA      = 0xBC
	VK_OEM_COPY       = 0xF2
	VK_OEM_CUSEL      = 0xEF
	VK_OEM_ENLW       = 0xF4
	VK_OEM_FINISH     = 0xF1
	VK_OEM_FJ_LOYA    = 0x95
	VK_OEM_FJ_MASSHOU = 0x93
	VK_OEM_FJ_ROYA    = 0x96
	VK_OEM_FJ_TOUROKU = 0x94
	VK_OEM_JUMP       = 0xEA
	VK_OEM_MINUS      = 0xBD
	VK_OEM_PA1        = 0xEB
	VK_OEM_PA2        = 0xEC
	VK_OEM_PA3        = 0xED
	VK_OEM_PERIOD     = 0xBE
	VK_OEM_PLUS       = 0xBB
	VK_OEM_RESET      = 0xE9
	VK_OEM_WSCTRL     = 0xEE
	VK_PA1            = 0xFD
	VK_PACKET         = 0xE7
	VK_PLAY           = 0xFA
	VK_PROCESSKEY     = 0xE5
	VK_RETURN         = 0x0D
	VK_SELECT         = 0x29
	VK_SEPARATOR      = 0x6C
	VK_SPACE          = 0x20
	VK_SUBTRACT       = 0x6D
	VK_TAB            = 0x09
	VK_ZOOM           = 0xFB
	VK_LCONTROL       = 0xA2
)

var keyboardTable [0xFB]int

// http://msdn.microsoft.com/en-us/library/windows/desktop/dd162805.aspx
type POINT struct {
	X, Y int32
}

// http://msdn.microsoft.com/en-us/library/windows/desktop/ms644958.aspx
type MSG struct {
	Hwnd    HWND
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      POINT
}

//https://docs.microsoft.com/en-us/windows/desktop/api/winuser/ns-winuser-tagkbdllhookstruct
type KBDLLHOOKSTRUCT struct {
	vkCode      DWORD
	scanCode    DWORD
	flags       DWORD
	time        DWORD
	dwExtraInfo ULONG_PTR
}
type HOOKPROC func(int, WPARAM, LPARAM) LRESULT
type WINEVENTPROC func(
	hWinEventHook HWINEVENTHOOK,
	event uint32,
	hwnd HWND,
	idObject int32,
	idChild int32,
	idEventThread uint32,
	dwmsEventTime uint32) uintptr

func SetWinEventHook(
	eventMin DWORD,
	eventMax DWORD,
	hmodWinEventProc HMODULE,
	pfnWinEventProc WINEVENTPROC,
	idProcess DWORD,
	idThread DWORD,
	dwFlags DWORD) HWINEVENTHOOK {

	pfnWinEventProcCallback := syscall.NewCallback(pfnWinEventProc)
	ret1, ret2, err := procSetWindowsHookEx.Call(
		uintptr(eventMin),
		uintptr(eventMax),
		uintptr(hmodWinEventProc),
		pfnWinEventProcCallback,
		uintptr(idProcess),
		uintptr(idThread),
		uintptr(dwFlags))
	fmt.Printf("%v %v\n", ret2, err)
	return HWINEVENTHOOK(ret1)
}

func SetWindowsHookEx(idHook int, lpfn HOOKPROC, hMod HINSTANCE, dwThreadId DWORD) HHOOK {
	ret, _, _ := procSetWindowsHookEx.Call(
		uintptr(idHook),
		uintptr(syscall.NewCallback(lpfn)),
		uintptr(hMod),
		uintptr(dwThreadId),
	)
	return HHOOK(ret)
}

func CallNextHookEx(hhk HHOOK, nCode int, wParam WPARAM, lParam LPARAM) LRESULT {
	ret, _, _ := procCallNextHookEx.Call(
		uintptr(hhk),
		uintptr(nCode),
		uintptr(wParam),
		uintptr(lParam),
	)
	return LRESULT(ret)
}
func UnhookWindowsHookEx(hhk HHOOK) bool {
	ret, _, _ := procUnhookWindowsHookEx.Call(
		uintptr(hhk),
	)
	return ret != 0
}
func PostThreadMessage(idThread DWORD, msg UINT, wParam WPARAM, lParam LPARAM) {
	ret1, ret2, err := procPostThreadMessage.Call(
		uintptr(idThread),
		uintptr(msg),
		uintptr(wParam),
		uintptr(lParam))
	if err != nil {
		fmt.Printf("%v %v err: %v\n", ret1, ret2, err)
	}
}
func PostQuitMessage(code int) {
	ret, ret2, err := procQuitMessage.Call(
		uintptr(code))
	fmt.Printf("ret\n", ret, ret2, err)
}
func GetCurrentThreadId() int {
	ret1, ret2, err := procGetCurrentThreadId.Call()
	fmt.Printf("%v %v %v\n", ret1, ret2, err)
	return int(ret1)
}
func GetMessage(msg *MSG, hwnd HWND, msgFilterMin uint32, msgFilterMax uint32) int {
	ret, _, _ := procGetMessage.Call(
		uintptr(unsafe.Pointer(msg)),
		uintptr(hwnd),
		uintptr(msgFilterMin),
		uintptr(msgFilterMax))
	return int(ret)
}

func TranslateMessage(msg *MSG) bool {
	ret, _, _ := procTranslateMessage.Call(
		uintptr(unsafe.Pointer(msg)))
	return ret != 0
}

func DispatchMessage(msg *MSG) uintptr {
	ret, _, _ := procDispatchMessage.Call(
		uintptr(unsafe.Pointer(msg)))
	return ret
}

func LowLevelKeyboardProc(nCode int, wParam WPARAM, lParam LPARAM) LRESULT {
	ret, _, _ := procLowLevelKeyboard.Call(
		uintptr(nCode),
		uintptr(wParam),
		uintptr(lParam),
	)
	return LRESULT(ret)
}

func (s *SHook) LowLevelKeyboardProcCallback(nCode int, wParam WPARAM, lParam LPARAM) LRESULT {
	if nCode == HC_ACTION {
		kbdstruct := (*KBDLLHOOKSTRUCT)(unsafe.Pointer(lParam))
		code := byte(kbdstruct.vkCode)
		switch wParam {
		case WM_KEYDOWN:
			keyboardTable[code] = 1
			//fmt.Printf("[Event] -> WM_KEYDOWN -> %q\n", code)
			switch code {
			case VK_LCONTROL:
				break
			case VK_KEY_Q:
				break
			case VK_KEY_V:
				if keyboardTable[VK_LCONTROL] == 1 {
					s.out <- &SEvent{
						Type: 2,
						Spec: 1,
					}
				}
				fmt.Println("[win32hook] -> CTRL+V")
				break
			case VK_KEY_C:
				if keyboardTable[VK_KEY_Q] == 1 && keyboardTable[VK_LCONTROL] == 1 {
					s.out <- &SEvent{
						Type: 3,
						Spec: 2,
					}
				} else if keyboardTable[VK_LCONTROL] == 1 {
					s.out <- &SEvent{
						Type: 1,
						Spec: 1,
					}
				}
				fmt.Println("[win32hook] -> CTRL+C")
				break
			}
			break
		case WM_KEYUP:
			keyboardTable[code] = 0
			//fmt.Printf("[Event] -> WM_KEYUP -> %q\n", code)
			break
		}
		//fmt.Printf("[Event] key pressed: %q\n", code)
	}
	return CallNextHookEx(keyboardHook, nCode, wParam, lParam)
}
func (s *SHook) hookStart() {
	fmt.Printf("win32hook -> started\n")
	keyboardHook = SetWindowsHookEx(WH_KEYBOARD_LL,
		s.LowLevelKeyboardProcCallback, 0, 0)
	var msg MSG
	s.threadId = GetCurrentThreadId()
	fmt.Printf("win32hook -> thread id: %d\n", s.threadId)
	fmt.Printf("win32hook -> starting [GetMessage]\n")
	s.start <- true
	for GetMessage(&msg, 0, 0, 0) != 0 {
	}
	fmt.Printf("win32hook -> stop signal, unloading hook\n")
	UnhookWindowsHookEx(keyboardHook)
	keyboardHook = 0
	fmt.Printf("win32hook -> unloaded\n")
	fmt.Printf("win32hook -> signalling close\n")
	s.start <- true
}

type SEvent struct {
	Type int
	Spec int
	buf  string
}
type SHook struct {
	out      chan *SEvent
	start    chan bool
	threadId int
}

func (s *SHook) Start(ev chan *SEvent) {
	fmt.Printf("[EventHook] -> starting thread\n")
	s.out = ev
	s.start = make(chan bool)
	go s.hookStart()
	<-s.start
	fmt.Printf("[EventHook] -> started\n")
}
func (s *SHook) Close() {
	fmt.Printf("[EventHook] -> closing thread\n")
	fmt.Printf("[EventHook] -> Signalling thread id: %d\n", s.threadId)
	PostThreadMessage(DWORD(s.threadId), 0x0012, 0, 0)
	<-s.start
	fmt.Printf("[EventHook] -> closed thread\n")
}
