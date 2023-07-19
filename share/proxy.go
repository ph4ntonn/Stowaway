package share

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"

	"Stowaway/utils"
)

type Proxy interface {
	Dial() (net.Conn, error)
}

type Socks5Proxy struct {
	PeerAddr  string
	ProxyAddr string
	UserName  string
	Password  string
}

func NewSocks5Proxy(peerAddr, proxyAddr, username, password string) *Socks5Proxy {
	proxy := new(Socks5Proxy)
	proxy.PeerAddr = peerAddr
	proxy.ProxyAddr = proxyAddr
	proxy.UserName = username
	proxy.Password = password
	return proxy
}

func (proxy *Socks5Proxy) Dial() (net.Conn, error) {
	var NOT_SUPPORT = errors.New("unknown protocol")
	var WRONG_AUTH = errors.New("wrong auth method")
	var SERVER_ERROR = errors.New("proxy server error")
	var TOO_LONG = errors.New("user/pass too long(max 255)")
	var AUTH_FAIL = errors.New("wrong username/password")

	proxyConn, err := net.Dial("tcp", proxy.ProxyAddr)
	if err != nil {
		return proxyConn, err
	}

	host, portS, err := net.SplitHostPort(proxy.PeerAddr)
	if err != nil {
		return proxyConn, err
	}
	portUint64, err := strconv.ParseUint(portS, 10, 16)
	if err != nil {
		return proxyConn, err
	}

	port := uint16(portUint64)
	portB := make([]byte, 2)
	binary.BigEndian.PutUint16(portB, port)
	// No Auth
	if proxy.UserName == "" && proxy.Password == "" {
		proxyConn.Write([]byte{0x05, 0x01, 0x00})
	} else {
		// u and p
		proxyConn.Write([]byte{0x05, 0x01, 0x02})
	}

	authWayBuf := make([]byte, 2)

	_, err = io.ReadFull(proxyConn, authWayBuf)
	if err != nil {
		return proxyConn, err
	}

	if authWayBuf[0] == 0x05 {
		switch authWayBuf[1] {
		case 0x00:
		case 0x02:
			userLen := len(proxy.UserName)
			passLen := len(proxy.Password)
			if userLen > 255 || passLen > 255 {
				return proxyConn, TOO_LONG
			}

			buff := make([]byte, 0, 3+userLen+passLen)
			buff = append(buff, 0x01, byte(userLen))
			buff = append(buff, []byte(proxy.UserName)...)
			buff = append(buff, byte(passLen))
			buff = append(buff, []byte(proxy.Password)...)
			proxyConn.Write(buff)

			responseBuf := make([]byte, 2)
			_, err = io.ReadFull(proxyConn, responseBuf)
			if err != nil {
				return proxyConn, err
			}

			if responseBuf[0] == 0x01 {
				if responseBuf[1] == 0x00 {
					break
				} else {
					return proxyConn, AUTH_FAIL
				}
			} else {
				return proxyConn, NOT_SUPPORT
			}
		case 0xff:
			return proxyConn, WRONG_AUTH
		default:
			return proxyConn, NOT_SUPPORT
		}

		var (
			buff []byte
			ip   net.IP
		)
		isV4 := utils.CheckIfIP4(host)
		if isV4 {
			buff = make([]byte, 0, 10)
			ip = net.ParseIP(host).To4()
			buff = append(buff, []byte{0x05, 0x01, 0x00, 0x01}...)
		} else {
			buff = make([]byte, 0, 22)
			ip = net.ParseIP(host).To16()
			buff = append(buff, []byte{0x05, 0x01, 0x00, 0x04}...)
		}
		buff = append(buff, []byte(ip)...)
		buff = append(buff, portB...)
		proxyConn.Write(buff)

		respBuf := make([]byte, 4)
		_, err = io.ReadFull(proxyConn, respBuf)
		if err != nil {
			return proxyConn, err
		}
		if respBuf[0] == 0x05 {
			if respBuf[1] != 0x00 {
				return proxyConn, SERVER_ERROR
			}
			switch respBuf[3] {
			case 0x01:
				resultBuf := make([]byte, 6)
				_, err = io.ReadFull(proxyConn, resultBuf)
			case 0x04:
				resultBuf := make([]byte, 18)
				_, err = io.ReadFull(proxyConn, resultBuf)
			default:
				return proxyConn, NOT_SUPPORT
			}
			if err != nil {
				return proxyConn, err
			}

			return proxyConn, nil
		} else {
			return proxyConn, NOT_SUPPORT
		}
	} else {
		return proxyConn, NOT_SUPPORT
	}
}

type HTTPProxy struct {
	PeerAddr  string
	ProxyAddr string
}

func NewHTTPProxy(peerAddr, proxyAddr string) *HTTPProxy {
	proxy := new(HTTPProxy)
	proxy.PeerAddr = peerAddr
	proxy.ProxyAddr = proxyAddr
	return proxy
}

func (proxy *HTTPProxy) Dial() (net.Conn, error) {
	var SERVER_ERROR = errors.New("proxy server error")
	var RESPONSE_TOO_LARGE = errors.New("http connect response is too large > 40KB")

	proxyConn, err := net.Dial("tcp", proxy.ProxyAddr)
	if err != nil {
		return proxyConn, SERVER_ERROR
	}

	var http_proxy_payload_template = "CONNECT %s HTTP/1.1\r\n" +
		"Content-Length: 0\r\n\r\n"
	var payload = fmt.Sprintf(http_proxy_payload_template, proxy.PeerAddr)
	var buf = []byte(payload)

	proxyConn.Write(buf)

	var done = "\r\n\r\n"
	var success = "HTTP/1.1 200"
	var begin = 0
	var resultBuf = make([]byte, 40960)

	for {
		count, err := proxyConn.Read(resultBuf[begin:])
		if err != nil {
			return proxyConn, SERVER_ERROR
		}

		begin += count
		if begin >= 40960 {
			return proxyConn, RESPONSE_TOO_LARGE
		}

		if string(resultBuf[begin-4:begin]) == done {
			if string(resultBuf[:len(success)]) == success {
				return proxyConn, nil
			}
			return proxyConn, SERVER_ERROR
		}
	}
}
