/*
 * @Author: ph4ntom
 * @Date: 2021-03-11 14:21:46
 * @LastEditors: ph4ntom
 * @LastEditTime: 2021-03-11 14:28:56
 */
package share

import (
	"Stowaway/utils"
	"errors"
	"io"
	"net"
	"time"
)

func ActivePreAuth(conn net.Conn, key string) (err error) {
	err = sendSecret(conn, key)
	err = checkSecret(conn, key)
	return
}

func PassivePreAuth(conn net.Conn, key string) (err error) {
	err = checkSecret(conn, key)
	err = sendSecret(conn, key)
	return
}

func sendSecret(conn net.Conn, key string) error {
	var NOT_VALID = errors.New("Not valid secret,check the secret!")

	defer conn.SetReadDeadline(time.Time{})
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))

	secret := utils.GetStringMd5(key)
	conn.Write([]byte(secret[:16]))

	buffer := make([]byte, 16)
	count, err := io.ReadFull(conn, buffer)

	if timeoutErr, ok := err.(net.Error); ok && timeoutErr.Timeout() {
		conn.Close()
		return NOT_VALID
	}

	if err != nil {
		conn.Close()
		return NOT_VALID
	}

	if string(buffer[:count]) == secret[:16] {
		return nil
	}

	conn.Close()

	return NOT_VALID
}

// CheckSecret 检查secret值，在连接建立前测试合法性
func checkSecret(conn net.Conn, key string) error {
	var NOT_VALID = errors.New("Not valid secret,check the secret!")

	defer conn.SetReadDeadline(time.Time{})
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))

	secret := utils.GetStringMd5(key)

	buffer := make([]byte, 16)
	count, err := io.ReadFull(conn, buffer)

	if timeoutErr, ok := err.(net.Error); ok && timeoutErr.Timeout() {
		conn.Close()
		return NOT_VALID
	}

	if err != nil {
		conn.Close()
		return NOT_VALID
	}

	if string(buffer[:count]) == secret[:16] {
		conn.Write([]byte(secret[:16]))
		return nil
	}

	conn.Close()

	return NOT_VALID
}
