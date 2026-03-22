//go:build darwin
package main

import (
	"os/exec"

	"fyne.io/fyne/v2"
)

func MaximizeWindow(w fyne.Window) {
	// Not supported natively yet on Mac via syscalls here, 
	// Fyne handles full screen if needed, but we'll leave it as no-op for now.
}

func OpenExplorer(path string) {
	exec.Command("open", "-R", path).Start()
}

func OpenFolder(path string) {
	exec.Command("open", path).Start()
}
