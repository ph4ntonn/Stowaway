package net

import (
	"crypto/tls"
	"fmt"
	"net"
)

func Listener(conn net.Conn, protocol string) (c net.Conn, err error) {
	switch protocol {
	case "tcp":
		return conn, nil
	default:
		return nil, fmt.Errorf("unsupport protocol: %s", protocol)
	}
}

func ListenerWithTLS(conn net.Conn, protocol string, tlsConfig *tls.Config) (c net.Conn, err error) {
	if tlsConfig != nil {
		conn = WrapTLSServerConn(conn, tlsConfig)
	}

	c, err = Listener(conn, protocol)

	return
}
