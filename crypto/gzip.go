package crypto

import (
	"bytes"
	"compress/gzip"
	"io/ioutil"
)

// Thx to code from @lz520520
func GzipCompress(src []byte) []byte {
	var in bytes.Buffer
	w := gzip.NewWriter(&in)
	w.Write(src)
	w.Close()
	return in.Bytes()
}

func GzipDecompress(src []byte) []byte {
	dst := make([]byte, 0)
	br := bytes.NewReader(src)
	gr, err := gzip.NewReader(br)
	if err != nil {
		return dst
	}
	defer gr.Close()
	tmp, err := ioutil.ReadAll(gr)
	if err != nil {
		return dst
	}
	dst = tmp
	return dst
}
