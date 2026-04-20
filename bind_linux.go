//go:build linux

package main

import (
	"net"
	"syscall"
)

// bindToDevice pins the UDP socket to a specific NIC via SO_BINDTODEVICE.
func bindToDevice(conn *net.UDPConn, nic string) error {
	raw, err := conn.SyscallConn()
	if err != nil {
		return err
	}
	var setErr error
	ctlErr := raw.Control(func(fd uintptr) {
		setErr = syscall.SetsockoptString(int(fd), syscall.SOL_SOCKET, syscall.SO_BINDTODEVICE, nic)
	})
	if ctlErr != nil {
		return ctlErr
	}
	return setErr
}
