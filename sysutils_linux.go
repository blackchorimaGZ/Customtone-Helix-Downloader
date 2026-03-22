//go:build linux
package main

import (
	"os/exec"

	"fyne.io/fyne/v2"
)

func MaximizeWindow(w fyne.Window) {
	// Fyne handles basic windowing on Linux, no simple universal syscall for this.
}

func OpenExplorer(path string) {
	// xdg-open doesn't always support selection, but it will open the folder.
	exec.Command("xdg-open", path).Start()
}

func OpenFolder(path string) {
	exec.Command("xdg-open", path).Start()
}
