package utils

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"math/rand"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"golang.org/x/text/encoding/simplifiedchinese"
)

func GenerateUUID() string {
	u2, _ := uuid.NewV4()
	uu := strings.Replace(u2.String(), "-", "", -1)
	uuid := uu[11:21]
	return uuid
}

func GetStringMd5(s string) string {
	md5 := md5.New()
	md5.Write([]byte(s))
	md5Str := hex.EncodeToString(md5.Sum(nil))
	return md5Str
}

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

func Str2Int(str string) (int, error) {
	num, err := strconv.ParseInt(str, 10, 32)
	return int(uint32(num)), err
}

func Int2Str(num int) string {
	b := strconv.Itoa(num)
	return b
}

func CheckSystem() (sysType uint32) {
	var os = runtime.GOOS
	switch os {
	case "windows":
		sysType = 0x01
	case "linux":
		sysType = 0x02
	default:
		sysType = 0x03
	}
	return
}

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
		err = errors.New("please input either port(1~65535) or ip:port(1-65535)")
		return
	}

	if err != nil || readyPort < 1 || readyPort > 65535 || readyIP == "" {
		err = errors.New("please input either port(1~65535) or ip:port(1-65535)")
		return
	}

	normalAddr = readyIP + ":" + strconv.Itoa(readyPort)
	reuseAddr = "0.0.0.0:" + strconv.Itoa(readyPort)

	return
}

func CheckIfIP4(ip string) bool {
	for i := 0; i < len(ip); i++ {
		switch ip[i] {
		case '.':
			return true
		case ':':
			return false
		}
	}
	return false
}

func CheckRange(nodes []int) {
	for m := len(nodes) - 1; m > 0; m-- {
		var flag bool = false
		for n := 0; n < m; n++ {
			if nodes[n] > nodes[n+1] {
				temp := nodes[n]
				nodes[n] = nodes[n+1]
				nodes[n+1] = temp
				flag = true
			}
		}
		if !flag {
			break
		}
	}
}

func GetDigitLen(num int) int {
	var length int
	for {
		num = num / 10
		if num != 0 {
			length++
		} else {
			length++
			return length
		}
	}
}

func GetRandomString(l int) string {
	str := "0123456789abcdefghijklmnopqrstuvwxyz"
	bytes := []byte(str)
	result := []byte{}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < l; i++ {
		result = append(result, bytes[r.Intn(len(bytes))])
	}
	return string(result)
}

func GetRandomInt(max int) int {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return r.Intn(max)
}

func ParseFileCommand(commands []string) (string, string, error) {
	if len(commands) == 2 {
		return commands[0], commands[1], nil
	} else if len(commands) > 2 {
		var count int
		full := strings.Join(commands, " ")

		for _, char := range full {
			if char == '"' {
				count++
			}
		}

		print("count is ", count)

		if count > 0 && count%2 == 0 {
			var final []string
			for _, part := range strings.Split(full, "\"") {
				if ready := strings.Trim(part, " \t\r\n"); ready != "" {
					final = append(final, ready)
				}
			}

			if len(final) == 2 {
				return final[0], final[1], nil
			} else {
				return "", "", errors.New("wrong format")
			}
		} else {
			return "", "", errors.New("wrong format")
		}
	}

	return "", "", errors.New("not enough arguments")
}

func ConvertStr2GBK(str string) string {
	ret, err := simplifiedchinese.GBK.NewEncoder().String(str)
	if err != nil {
		ret = str
	}
	return ret
}

func ConvertGBK2Str(gbkStr string) string {
	ret, err := simplifiedchinese.GBK.NewDecoder().String(gbkStr)
	if err != nil {
		ret = gbkStr
	}
	return ret

}
