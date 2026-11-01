//go:build !windows

package app

import "os/exec"

// hideWindow does nothing where programs have no console window of their
// own.
func hideWindow(cmd *exec.Cmd) {}
