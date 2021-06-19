package crypto

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"errors"
)

func KeyPadding(key []byte) ([]byte, error) {
	// if no key,just return
	if string(key) == "" {
		return nil, nil
	}
	// if key is set,padding it
	keyLength := float32(len(key))
	if keyLength/8 >= 4 {
		return nil, errors.New("Key too long! Should shorter than 32 bytes")
	}
	padding := 32 - len(key)
	padText := bytes.Repeat([]byte{byte(0)}, padding)
	return append(key, padText...), nil
}

func AESDecrypt(cryptedData, key []byte) []byte {
	if key == nil {
		return cryptedData
	}

	block, _ := aes.NewCipher(key)
	blockSize := block.BlockSize()
	blockMode := cipher.NewCBCDecrypter(block, key[:blockSize])
	origData := make([]byte, len(cryptedData))
	blockMode.CryptBlocks(origData, cryptedData)
	origData = PKCS7UnPadding(origData)
	return origData
}

func PKCS7UnPadding(origData []byte) []byte {
	length := len(origData)
	unpadding := int(origData[length-1])
	return origData[:length-unpadding]
}

func AESEncrypt(origData, key []byte) []byte {
	if key == nil {
		return origData
	}

	block, _ := aes.NewCipher(key)
	origData = PKCS7Padding(origData, block.BlockSize())
	blockMode := cipher.NewCBCEncrypter(block, key[:block.BlockSize()])
	crypted := make([]byte, len(origData))
	blockMode.CryptBlocks(crypted, origData)
	return crypted
}

func PKCS7Padding(origData []byte, blockSize int) []byte {
	padding := blockSize - len(origData)%blockSize
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(origData, padText...)
}
