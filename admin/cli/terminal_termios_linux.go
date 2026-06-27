//go:build linux
// +build linux

package cli

import "golang.org/x/sys/unix"

func ioctlReadTermios() uint {
	return unix.TCGETS
}

func ioctlWriteTermios() uint {
	return unix.TCSETS
}
