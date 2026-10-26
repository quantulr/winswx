//go:build windows

package main

import (
	"os/exec"
	"syscall"
)

// createNoWindow is CREATE_NO_WINDOW: run a console program without a
// console window of its own.
const createNoWindow = 0x08000000

// hideWindow runs cmd without a console window. The app is a GUI program,
// and WinSW is a console program whose black window would otherwise flash
// on every status query and service command.
func hideWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: createNoWindow,
	}
}
