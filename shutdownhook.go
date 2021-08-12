package shutdownhook

import (
	"fmt"
	"runtime"
	"sync"
	"syscall"
	"unsafe"
)

func New(hook func()) error {
	// Call win32 API from a single OS thread.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	cls, err := registerClassEx(&wndClassEx{
		cbSize:        uint32(unsafe.Sizeof(wndClassEx{})),
		lpfnWndProc:   syscall.NewCallback(newWindowsProc(hook)),
		lpszClassName: syscall.StringToUTF16Ptr("ShutDownHook"),
	})
	if err != nil {
		return err
	}
	hwnd, err := createWindowEx(
		_WS_EX_APPWINDOW|_WS_EX_WINDOWEDGE,
		cls,
		"ShutDownHook",
		_WS_OVERLAPPEDWINDOW,
		_CW_USEDEFAULT,
		_CW_USEDEFAULT,
		800,
		600,
		0,
		0,
		0,
		0)
	if err != nil {
		return err
	}
	updateWindow(hwnd)
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		for {
			var m msg
			if getMessage(&m, hwnd, 0, 0) > 0 {
				translateMessage(&m)
				dispatchMessage(&m)
			} else {
				break
			}
		}
	}()
	return nil
}

var (
	user32            = syscall.NewLazyDLL("user32.dll")
	_CreateWindowEx   = user32.NewProc("CreateWindowExW")
	_DefWindowProc    = user32.NewProc("DefWindowProcW")
	_DispatchMessage  = user32.NewProc("DispatchMessageW")
	_GetMessage       = user32.NewProc("GetMessageW")
	_TranslateMessage = user32.NewProc("TranslateMessage")
	_RegisterClassExW = user32.NewProc("RegisterClassExW")
	_UpdateWindow     = user32.NewProc("UpdateWindow")
)

type wndClassEx struct {
	cbSize        uint32
	style         uint32
	lpfnWndProc   uintptr
	cbClsExtra    int32
	cbWndExtra    int32
	hInstance     syscall.Handle
	hIcon         syscall.Handle
	hCursor       syscall.Handle
	hbrBackground syscall.Handle
	lpszMenuName  *uint16
	lpszClassName *uint16
	hIconSm       syscall.Handle
}

type msg struct {
	hwnd     syscall.Handle
	message  uint32
	wParam   uintptr
	lParam   uintptr
	time     uint32
	pt       point
	lPrivate uint32
}

type point struct {
	x, y int32
}

const (
	_CW_USEDEFAULT = -0x80000000

	_WS_OVERLAPPED       = 0x00000000
	_WS_CAPTION          = 0x00C00000
	_WS_SYSMENU          = 0x00080000
	_WS_THICKFRAME       = 0x00040000
	_WS_MINIMIZEBOX      = 0x00020000
	_WS_MAXIMIZEBOX      = 0x00010000
	_WS_OVERLAPPEDWINDOW = _WS_OVERLAPPED | _WS_CAPTION | _WS_SYSMENU | _WS_THICKFRAME | _WS_MINIMIZEBOX | _WS_MAXIMIZEBOX

	_WS_EX_APPWINDOW  = 0x00040000
	_WS_EX_WINDOWEDGE = 0x00000100

	_WM_QUERYENDSESSION = 17
	_WM_ENDSESSION      = 22
)

func newWindowsProc(f func()) func(hwnd syscall.Handle, msg uint32, wParam, lParam uintptr) uintptr {
	var once sync.Once
	return func(hwnd syscall.Handle, msg uint32, wParam, lParam uintptr) uintptr {
		switch msg {
		case _WM_QUERYENDSESSION, _WM_ENDSESSION:
			once.Do(f)
			return 1
		default:
			return defWindowProc(hwnd, msg, wParam, lParam)
		}
	}
}

func createWindowEx(dwExStyle uint32,
	lpClassName uint16, lpWindowName string,
	dwStyle uint32,
	x, y, w, h int32,
	hWndParent, hMenu, hInstance syscall.Handle,
	lpParam uintptr,
) (syscall.Handle, error) {
	hwnd, _, err := _CreateWindowEx.Call(
		uintptr(dwExStyle),
		uintptr(lpClassName),
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(lpWindowName))),
		uintptr(dwStyle),
		uintptr(x), uintptr(y),
		uintptr(w), uintptr(h),
		uintptr(hWndParent),
		uintptr(hMenu),
		uintptr(hInstance),
		uintptr(lpParam))
	if hwnd == 0 {
		return 0, fmt.Errorf("CreateWindowEx failed: %v", err)
	}
	return syscall.Handle(hwnd), nil
}

func defWindowProc(hwnd syscall.Handle, msg uint32, wparam, lparam uintptr) uintptr {
	r, _, _ := _DefWindowProc.Call(uintptr(hwnd), uintptr(msg), wparam, lparam)
	return r
}

func getMessage(m *msg, hwnd syscall.Handle, wMsgFilterMin, wMsgFilterMax uint32) int32 {
	r, _, _ := _GetMessage.Call(
		uintptr(unsafe.Pointer(m)),
		uintptr(hwnd),
		uintptr(wMsgFilterMin),
		uintptr(wMsgFilterMax),
	)
	return int32(r)
}

func translateMessage(m *msg) {
	_, _, _ = _TranslateMessage.Call(uintptr(unsafe.Pointer(m)))
}

func dispatchMessage(m *msg) {
	_, _, _ = _DispatchMessage.Call(uintptr(unsafe.Pointer(m)))
}

func registerClassEx(cls *wndClassEx) (uint16, error) {
	a, _, err := _RegisterClassExW.Call(uintptr(unsafe.Pointer(cls)))
	if a == 0 {
		return 0, fmt.Errorf("RegisterClassExW failed: %v", err)
	}
	return uint16(a), nil
}

func updateWindow(hwnd syscall.Handle) {
	_, _, _ = _UpdateWindow.Call(uintptr(hwnd))
}
