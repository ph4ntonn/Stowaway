package initial

import (
	"Stowaway/utils"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"strconv"
)

type Proxy struct {
	AdminAddr string
	ProxyAddr string
	UserName  string
	Password  string
}

func NewProxy(adminAddr, proxyAddr, username, password string) *Proxy {
	proxy := new(Proxy)
	proxy.AdminAddr = adminAddr
	proxy.ProxyAddr = proxyAddr
	proxy.UserName = username
	proxy.Password = password
	return proxy
}

func (proxy *Proxy) Dial() (net.Conn, error) {
	var NOT_SUPPORT = errors.New("Unknown protocol")
	var NEED_AUTH = errors.New("Proxy server need username/password")
	var SERVER_ERROR = errors.New("Proxy server error")
	var TOO_LONG = errors.New("User/Pass too long(max 255)")
	var AUTH_FAIL = errors.New("Wrong username/password")

	proxyConn, err := net.Dial("tcp", proxy.ProxyAddr)
	if err != nil {
		return proxyConn, err
	}

	host, portS, err := net.SplitHostPort(proxy.AdminAddr)
	if err != nil {
		return proxyConn, err
	}
	portUint64, err := strconv.ParseUint(portS, 10, 16)
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
			return proxyConn, NEED_AUTH
		default:
			return proxyConn, NOT_SUPPORT
		}

		isV4 := utils.CheckIfIP4(host)
		if isV4 {
			buff := make([]byte, 0, 10)
			ip := net.ParseIP(host).To4()
			buff = append(buff, []byte{0x05, 0x01, 0x00, 0x01}...)
			buff = append(buff, []byte(ip)...)
			buff = append(buff, portB...)
			proxyConn.Write(buff)
		} else {
			buff := make([]byte, 0, 22)
			ip := net.ParseIP(host).To16()
			buff = append(buff, []byte{0x05, 0x01, 0x00, 0x04}...)
			buff = append(buff, []byte(ip)...)
			buff = append(buff, portB...)
		}

		respBuf := make([]byte, 4)
		_, err = io.ReadFull(proxyConn, respBuf)
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
			return proxyConn, nil
		} else {
			return proxyConn, NOT_SUPPORT
		}
	} else {
		return proxyConn, NOT_SUPPORT
	}
}
