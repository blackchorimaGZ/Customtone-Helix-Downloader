//go:build !windows
package main

import (
	"os/exec"

	"fyne.io/fyne/v2"
)

func MaximizeWindow(w fyne.Window) {
	// No-op for macOS/Linux unless we use specific AppleScript or similar
}

func OpenExplorer(path string) {
	exec.Command("open", "-R", path).Start()
}

func OpenFolder(path string) {
	exec.Command("open", path).Start()
}
