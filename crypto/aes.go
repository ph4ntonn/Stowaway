package crypto

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"errors"
)

// KeyPadding 补齐密钥长度至32字节
func KeyPadding(key []byte) ([]byte, error) {
	keyLength := float32(len(key))
	if keyLength/8 >= 4 {
		return nil, errors.New("Key too long! Should shorter than 32 bytes")
	}
	padding := 32 - len(key)
	padText := bytes.Repeat([]byte{byte(0)}, padding)
	return append(key, padText...), nil
}

// AESDecrypt 解密
func AESDecrypt(crypted, key []byte) []byte {
	block, _ := aes.NewCipher(key)
	blockSize := block.BlockSize()
	blockMode := cipher.NewCBCDecrypter(block, key[:blockSize])
	origData := make([]byte, len(crypted))
	blockMode.CryptBlocks(origData, crypted)
	origData = PKCS7UnPadding(origData)
	return origData
}

// PKCS7UnPadding 去补码
func PKCS7UnPadding(origData []byte) []byte {
	length := len(origData)
	unpadding := int(origData[length-1])
	return origData[:length-unpadding]
}

// AESEncrypt 加密
func AESEncrypt(origData, key []byte) []byte {
	//获取block块
	block, _ := aes.NewCipher(key)
	//补码
	origData = PKCS7Padding(origData, block.BlockSize())
	//加密模式，
	blockMode := cipher.NewCBCEncrypter(block, key[:block.BlockSize()])
	//创建明文长度的数组
	crypted := make([]byte, len(origData))
	//加密明文
	blockMode.CryptBlocks(crypted, origData)
	return crypted
}

// PKCS7Padding 补码
func PKCS7Padding(origData []byte, blockSize int) []byte {
	//计算需要补几位数
	padding := blockSize - len(origData)%blockSize
	println("blocksize is ",blockSize,"padding is ",padding)
	//在切片后面追加char数量的byte(char)
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(origData, padText...)
}
