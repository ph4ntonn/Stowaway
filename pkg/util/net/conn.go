package net

import (
	"crypto/tls"
	"fmt"
	"net"
	"time"
)

func ConnectServerByProxy(proxyURL string, protocol string, addr string, tlsConfig *tls.Config, domainAddr string) (c net.Conn, err error) {
	switch protocol {
	case "tcp":
		c, err = net.DialTimeout("tcp", addr, 10*time.Second)
		if tlsConfig != nil {
			c = WrapTLSClientConn(c, tlsConfig)
		}
		return
	default:
		return nil, fmt.Errorf("unsupport protocol: %s", protocol)
	}
}

func ConnectServerByProxyWithTLS(proxyURL string, protocol string, addr string, tlsConfig *tls.Config, domainAddr string) (c net.Conn, err error) {
	c, err = ConnectServerByProxy(proxyURL, protocol, addr, tlsConfig, domainAddr)

	return
}

func CloseConnSafe(conn net.Conn) {
	if conn != nil {
		switch conn.(type) {
		case *tls.Conn:
			tlsConn := conn.(*tls.Conn)
			if tlsConn.ConnectionState().HandshakeComplete {
				tlsConn.Close()
			}
		default:
			conn.Close()
		}

	}
}
