//go:build windows
package main

import (
	"os/exec"
	"path/filepath"
	"syscall"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver"
)

func MaximizeWindow(w fyne.Window) {
	if nw, ok := w.(driver.NativeWindow); ok {
		nw.RunNative(func(ctx interface{}) {
			if winCtx, ok := ctx.(*driver.WindowsWindowContext); ok {
				hwnd := winCtx.HWND
				// SW_MAXIMIZE is 3
				user32 := syscall.NewLazyDLL("user32.dll")
				showWindow := user32.NewProc("ShowWindow")
				showWindow.Call(hwnd, 3)
			}
		})
	}
}

func OpenExplorer(path string) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path
	}
	exec.Command("explorer", "/select,", absPath).Start()
}

func OpenFolder(path string) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path
	}
	exec.Command("explorer", absPath).Start()
}
