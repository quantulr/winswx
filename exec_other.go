//go:build !windows

package main

import "os/exec"

// hideWindow does nothing where programs have no console window of their
// own.
func hideWindow(cmd *exec.Cmd) {}
