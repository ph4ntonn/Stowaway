package protocol

import (
	"bytes"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"regexp"
	"strings"
	"time"
)

// TODO: The WebSocket data frames still have some issues See: https://datatracker.ietf.org/doc/html/rfc6455#section-5.
// But in actual testing, the NGINX reverse proxy works fine. Let's temporarily enable it, and if any issues arise, we can make improvements later.
const websocketGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
const websocketPath = "/deadbeef"

type WSProto struct {
	domain string
	conn   net.Conn
	*RawProto
}

func (proto *WSProto) CNegotiate() error {
	defer proto.conn.SetReadDeadline(time.Time{})
	proto.conn.SetReadDeadline(time.Now().Add(10 * time.Second))

	nonce := generateNonce()
	expectedAccept, err := getNonceAccept(nonce)
	if err != nil {
		return err
	}

	// 发送websocket头
	wsHeaders := fmt.Sprintf(`GET %s HTTP/1.1
Host: %s
Upgrade: websocket
Connection: Upgrade
Sec-WebSocket-Key: %s
Origin: https://google.com
Sec-WebSocket-Version: 13

`, websocketPath, proto.domain, nonce)

	wsHeaders = strings.ReplaceAll(wsHeaders, "\n", "\r\n")
	proto.conn.Write([]byte(wsHeaders))

	var ptr int
	result := bytes.Buffer{}
	buf := make([]byte, 1024)

	for {
		count, err := proto.conn.Read(buf)
		if err != nil {
			if err == io.EOF && count > 0 {
				result.Write(buf[:count])
			} else if timeoutErr, ok := err.(net.Error); ok && timeoutErr.Timeout() {
				return err
			}
			break
		}

		if count > 0 {
			result.Write(buf[ptr : ptr+count])
			ptr += count
			if bytes.HasSuffix(buf[:ptr], []byte("\r\n\r\n")) {
				break
			}
		}
	}

	resp := result.String()
	if !strings.Contains(resp, "Upgrade: websocket") ||
		!strings.Contains(resp, "Connection: Upgrade") ||
		!strings.Contains(resp, "Sec-WebSocket-Accept: "+string(expectedAccept)) {
		return errors.New("not websocket protocol")
	}

	return nil
}

func (proto *WSProto) SNegotiate() error {
	defer proto.conn.SetReadDeadline(time.Time{})
	proto.conn.SetReadDeadline(time.Now().Add(10 * time.Second))

	var ptr int
	result := bytes.Buffer{}
	buf := make([]byte, 1024)

	for {
		count, err := proto.conn.Read(buf)
		if err != nil {
			if err == io.EOF && count > 0 {
				result.Write(buf[:count])
			} else if timeoutErr, ok := err.(net.Error); ok && timeoutErr.Timeout() {
				return err
			}
			break
		}

		if count > 0 {
			result.Write(buf[ptr : ptr+count])
			ptr += count
			if bytes.HasSuffix(buf[:ptr], []byte("\r\n\r\n")) {
				break
			}
		}
	}

	re := regexp.MustCompile(`Sec-WebSocket-Key: (.*)`)
	tkey := re.FindStringSubmatch(strings.ReplaceAll(result.String(), "\r\n", "\n"))
	if len(tkey) < 2 {
		return errors.New("Sec-Websocket-Key is not in header")
	}

	key := tkey[1]
	expectedAccept, err := getNonceAccept([]byte(key))
	if err != nil {
		return err
	}

	respHeaders := fmt.Sprintf(`HTTP/1.1 101 Switching Protocols
Connection: Upgrade
Upgrade: websocket
Sec-WebSocket-Accept: %s

`, expectedAccept)

	respHeaders = strings.ReplaceAll(respHeaders, "\n", "\r\n")
	proto.conn.Write([]byte(respHeaders))
	return nil
}

type WSMessage struct {
	*RawMessage
}

func generateNonce() (nonce []byte) {
	key := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		panic(err)
	}
	nonce = make([]byte, 24)
	base64.StdEncoding.Encode(nonce, key)
	return
}

func getNonceAccept(nonce []byte) (expected []byte, err error) {
	h := sha1.New()
	if _, err = h.Write(nonce); err != nil {
		return
	}
	if _, err = h.Write([]byte(websocketGUID)); err != nil {
		return
	}
	expected = make([]byte, 28)
	base64.StdEncoding.Encode(expected, h.Sum(nil))
	return
}
