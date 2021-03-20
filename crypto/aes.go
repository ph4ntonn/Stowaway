/*
 * @Author: ph4ntom
 * @Date: 2021-03-08 19:15:11
 * @LastEditors: ph4ntom
 * @LastEditTime: 2021-03-20 15:23:19
 */
package crypto

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"errors"
)

/**
 * @description: Padding key to 32 bytes
 * @param {[]byte} key
 * @return {*}
 */
func KeyPadding(key []byte) ([]byte, error) {
	// if no key,just return
	if key == nil {
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

/**
 * @description: Decrypt data
 * @param {*} cryptedData
 * @param {[]byte} key
 * @return {*}
 */
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

/**
 * @description: Unpadding data -- follow PKCS7 rules
 * @param {[]byte} origData
 * @return {*}
 */
func PKCS7UnPadding(origData []byte) []byte {
	length := len(origData)
	unpadding := int(origData[length-1])
	return origData[:length-unpadding]
}

/**
 * @description: Encrypt clear data
 * @param {*} origData
 * @param {[]byte} key
 * @return {*}
 */
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

/**
 * @description: Padding data -- follow PKCS7 rules
 * @param {[]byte} origData
 * @param {int} blockSize
 * @return {*}
 */
func PKCS7Padding(origData []byte, blockSize int) []byte {
	padding := blockSize - len(origData)%blockSize
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(origData, padText...)
}
