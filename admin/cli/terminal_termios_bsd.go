//go:build darwin || freebsd
// +build darwin freebsd

package cli

import "golang.org/x/sys/unix"

func ioctlReadTermios() uint {
	return unix.TIOCGETA
}

func ioctlWriteTermios() uint {
	return unix.TIOCSETA
}
