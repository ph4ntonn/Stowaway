package http

import (
	"bufio"
	"net/http"
	"net/textproto"
	"strings"
)

func ParseRequest(s string) (*http.Request, error) {
	reader := bufio.NewReader(strings.NewReader(s))
	return http.ReadRequest(reader)
}

func ParseHeader(s string) http.Header {
	reader := bufio.NewReader(strings.NewReader(s + "\r\n\r\n"))
	tp := textproto.NewReader(reader)

	mimeHeader, err := tp.ReadMIMEHeader()
	if err != nil {
		return nil
	}

	// lzhttp.Header and lztextproto.MIMEHeader are both just a map[string][]string
	httpHeader := http.Header(mimeHeader)
	return httpHeader
}
