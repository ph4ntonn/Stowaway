package share

import (
	"errors"
	"io"
	"net"
	"time"

	"Stowaway/utils"
)

var AuthToken string

func GeneratePreAuthToken(secret string) {
	token := utils.GetStringMd5(secret)
	AuthToken = token[:16]
}

func ActivePreAuth(conn net.Conn) error {
	var NOT_VALID = errors.New("invalid secret, check the secret")
	var TIMEOUT = errors.New("connection timeout")

	defer conn.SetReadDeadline(time.Time{})
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))

	conn.Write([]byte(AuthToken))

	buffer := make([]byte, 16)
	count, err := io.ReadFull(conn, buffer)

	if timeoutErr, ok := err.(net.Error); ok && timeoutErr.Timeout() {
		conn.Close()
		return TIMEOUT
	}

	if err != nil {
		conn.Close()
		return errors.New(err.Error())
	}

	if string(buffer[:count]) == AuthToken {
		return nil
	}

	conn.Close()

	return NOT_VALID
}

func PassivePreAuth(conn net.Conn) error {
	var NOT_VALID = errors.New("invalid secret, check the secret")
	var TIMEOUT = errors.New("connection timeout")

	defer conn.SetReadDeadline(time.Time{})
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))

	buffer := make([]byte, 16)
	count, err := io.ReadFull(conn, buffer)

	if timeoutErr, ok := err.(net.Error); ok && timeoutErr.Timeout() {
		conn.Close()
		return TIMEOUT
	}

	if err != nil {
		conn.Close()
		return errors.New(err.Error())
	}

	if string(buffer[:count]) == AuthToken {
		conn.Write([]byte(AuthToken))
		return nil
	}

	conn.Close()

	return NOT_VALID
}
