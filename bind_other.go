//go:build !linux

package main

import "net"

// bindToDevice is a no-op on non-Linux platforms — SO_BINDTODEVICE doesn't
// exist there. The socket stays bound to 0.0.0.0 on all interfaces; the
// source-IP filter on the PDU is the only gate in that case.
func bindToDevice(_ *net.UDPConn, _ string) error {
	return nil
}
