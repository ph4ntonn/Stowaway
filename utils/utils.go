/*
 * @Author: ph4ntom
 * @Date: 2021-03-09 18:29:02
 * @LastEditors: ph4ntom
 * @LastEditTime: 2021-03-20 16:00:34
 */

package utils

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/gofrs/uuid"
)

// GenerateNodeID 生成一个nodeid
func GenerateUUID() string {
	u2, _ := uuid.NewV4()
	uu := strings.Replace(u2.String(), "-", "", -1)
	uuid := uu[11:21] //取10位，尽量减少包头长度
	return uuid
}

// GetStringMd5 生成md5值
func GetStringMd5(s string) string {
	md5 := md5.New()
	md5.Write([]byte(s))
	md5Str := hex.EncodeToString(md5.Sum(nil))
	return md5Str
}

// StringSliceReverse 倒置[]string
func StringSliceReverse(src []string) {
	if src == nil {
		return
	}
	count := len(src)
	mid := count / 2
	for i := 0; i < mid; i++ {
		tmp := src[i]
		src[i] = src[count-1]
		src[count-1] = tmp
		count--
	}
}

// StrUint32 string转换至uint32
func Str2Int(str string) (int, error) {
	num, err := strconv.ParseInt(str, 10, 32)
	return int(uint32(num)), err
}

func Int2Str(num int) string {
	b := strconv.Itoa(num)
	return b
}

// CheckSystem 检查所在的操作系统
func CheckSystem() (sysType uint32) {
	var os = runtime.GOOS
	switch os {
	case "windows":
		sysType = 0x01
	default:
		sysType = 0xff
	}
	return
}

// GetInfoViaSystem 获得系统信息
func GetSystemInfo() (string, string) {
	var os = runtime.GOOS
	switch os {
	case "windows":
		fallthrough
	case "linux":
		fallthrough
	case "darwin":
		hostname, err := exec.Command("hostname").Output()
		if err != nil {
			hostname = []byte("Null")
		}
		username, err := exec.Command("whoami").Output()
		if err != nil {
			username = []byte("Null")
		}

		fHostname := strings.TrimRight(string(hostname), " \t\r\n")
		fUsername := strings.TrimRight(string(username), " \t\r\n")

		return fHostname, fUsername
	default:
		return "NULL", "NULL"
	}
}

// CheckIPPort检查输入ip+port是否合法
func CheckIPPort(info string) (normalAddr string, reuseAddr string, err error) {
	var (
		readyIP   string
		readyPort int
	)

	spliltedInfo := strings.Split(info, ":")

	if len(spliltedInfo) == 1 {
		readyIP = "0.0.0.0"
		readyPort, err = strconv.Atoi(info)
	} else if len(spliltedInfo) == 2 {
		readyIP = spliltedInfo[0]
		readyPort, err = strconv.Atoi(spliltedInfo[1])
	} else {
		err = errors.New("Please input either port(1~65535) or ip:port(1-65535)!")
		return
	}

	if err != nil || readyPort < 1 || readyPort > 65535 || readyIP == "" {
		err = errors.New("Please input either port(1~65535) or ip:port(1-65535)!")
		return
	}

	normalAddr = readyIP + ":" + strconv.Itoa(readyPort)
	reuseAddr = "0.0.0.0:" + strconv.Itoa(readyPort)

	return
}
