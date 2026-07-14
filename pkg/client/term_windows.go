//go:build windows

package client

import (
	"os"
	"syscall"
	"unsafe"
)

// getTerminalWidth uses Windows console API to detect terminal width.
func getTerminalWidth() (int, bool) {
	fd := os.Stdout.Fd()
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	getConsoleScreenBufferInfo := kernel32.NewProc("GetConsoleScreenBufferInfo")

	type coord struct{ x, y int16 }
	type smallRect struct{ left, top, right, bottom int16 }
	type consoleScreenBufferInfo struct {
		size      coord
		cursorPos coord
		attrs     uint16
		window    smallRect
		maxWindow coord
	}

	var info consoleScreenBufferInfo
	ret, _, _ := getConsoleScreenBufferInfo.Call(uintptr(fd), uintptr(unsafe.Pointer(&info)))
	if ret == 0 {
		return 80, false
	}
	width := int(info.window.right - info.window.left + 1)
	if width <= 0 {
		width = 80
	}
	return width, true
}
