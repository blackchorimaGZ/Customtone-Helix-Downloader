//go:build linux
package main

import (
	"os/exec"

	"fyne.io/fyne/v2"
)

func MaximizeWindow(w fyne.Window) {
	// No-op for Linux
}

func OpenExplorer(path string) {
	exec.Command("xdg-open", path).Start()
}

func OpenFolder(path string) {
	exec.Command("xdg-open", path).Start()
}
