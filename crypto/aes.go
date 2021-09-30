package crypto

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"
)

func KeyPadding(key []byte) []byte {
	// if no key,just return
	if string(key) == "" {
		return nil
	}
	// if key is set, pad it
	keyLength := len(key)
	if keyLength > 32 {
		return key[:32]
	}
	padding := 32 - keyLength
	padText := bytes.Repeat([]byte{byte(0)}, padding)
	return append(key, padText...)
}

func genNonce(nonceSize int) []byte {
	nonce := make([]byte, nonceSize)
	io.ReadFull(rand.Reader, nonce)
	return nonce
}

func AESDecrypt(cryptedData, key []byte) []byte {
	if key == nil {
		return cryptedData
	}

	block, _ := aes.NewCipher(key)
	gcm, _ := cipher.NewGCM(block)
	nonceSize := gcm.NonceSize()
	nonce, cryptedData := cryptedData[:nonceSize], cryptedData[nonceSize:]
	origData, _ := gcm.Open(nil, nonce, cryptedData, nil)
	return origData
}

func AESEncrypt(origData, key []byte) []byte {
	if key == nil {
		return origData
	}

	block, _ := aes.NewCipher(key)
	gcm, _ := cipher.NewGCM(block)
	nonce := genNonce(gcm.NonceSize())
	return gcm.Seal(nonce, nonce, origData, nil)
}
