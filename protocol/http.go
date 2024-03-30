package protocol

// http.go is deprecated
// Just keep this for sample

import (
	"fmt"
	"io"

	"Stowaway/utils"
)

type HTTPProto struct{}

type HTTPMessage struct {
	HTTPHeader []byte
	*RawMessage
}

func (proto *HTTPProto) CNegotiate() error { return nil }

func (proto *HTTPProto) SNegotiate() error { return nil }

var partOne []string = []string{
	"POST /message/%s?number=%d&length=%d",
	"POST /uploads/%s?number=%d&length=%d",
	"POST /request/%s?number=%d&length=%d",
	"POST /hellowd/%s?number=%d&length=%d",
}

func (message *HTTPMessage) ConstructHeader() {
	reqHeaderPartOne := partOne[utils.GetRandomInt(4)]

	reqHeaderPartTwo := " HTTP/1.1\r\n" +
		"User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/89.0.4389.114 Safari/537.36\r\n" +
		"Host: www.google.com\r\n" +
		"Accept-Language: en-US,en;q=0.9,zh-CN;q=0.8,zh;q=0.7\r\n" +
		"Accept-Encoding: gzip, deflate, br\r\n" +
		"Accept: text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9\r\n" +
		"Content-Length: %d\r\n" +
		"\r\n"

	dataLen := len(message.RawMessage.DataBuffer)
	headerLen := len(message.RawMessage.HeaderBuffer)

	partTwoHeader := fmt.Sprintf(reqHeaderPartTwo, dataLen+headerLen)
	partTwoHeaderLen := len(partTwoHeader)

	partOneHeader := fmt.Sprintf(reqHeaderPartOne, utils.GetRandomString(6), utils.GetDigitLen(partTwoHeaderLen), partTwoHeaderLen)

	message.HTTPHeader = []byte(partOneHeader + partTwoHeader)
}

func (message *HTTPMessage) DeconstructHeader() {
	uselessBuf := make([]byte, 28)
	io.ReadFull(message.RawMessage.Conn, uselessBuf)

	numberBuf := make([]byte, 1)
	io.ReadFull(message.RawMessage.Conn, numberBuf)

	number, _ := utils.Str2Int(string(numberBuf))

	uselessBuf = make([]byte, 8)
	io.ReadFull(message.RawMessage.Conn, uselessBuf)

	lengthBuf := make([]byte, number)
	io.ReadFull(message.RawMessage.Conn, lengthBuf)

	length, _ := utils.Str2Int(string(lengthBuf))

	contentBuf := make([]byte, length)
	io.ReadFull(message.RawMessage.Conn, contentBuf)
}

func (message *HTTPMessage) SendMessage() {
	finalBuffer := append(message.HTTPHeader, message.HeaderBuffer...)
	finalBuffer = append(finalBuffer, message.DataBuffer...)
	message.RawMessage.Conn.Write(finalBuffer)
	// Don't forget to set both Buffer to nil!!!
	message.HeaderBuffer = nil
	message.DataBuffer = nil
	message.HTTPHeader = nil
}
