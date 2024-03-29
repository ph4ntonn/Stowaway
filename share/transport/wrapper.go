package transport

import (
	"crypto/tls"
	"net"
)

func WrapTLSClientConn(c net.Conn, tlsConfig *tls.Config) (out net.Conn) {
	out = tls.Client(c, tlsConfig)
	return
}

func WrapTLSServerConn(c net.Conn, tlsConfig *tls.Config) (out net.Conn) {
	out = tls.Server(c, tlsConfig)
	return
}
