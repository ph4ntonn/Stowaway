package common

import (
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"time"
)

/*-------------------------上传/下载文件相关代码--------------------------*/
func UploadFile(filename string, controlConn *net.Conn, nodeid uint32, getName chan bool, AESKey []byte, currentid uint32, Notagent bool) {
	var getresp bool = false
	var slicenum int = 0
	go CountDown(&getresp, getName)

	info, err := os.Stat(filename)
	if err != nil {
		getresp = true
		if Notagent {
			fmt.Println("[*]File not found!")
		} else {
			respData, _ := ConstructPayload(nodeid, "COMMAND", "FILENOTEXIST", " ", filename, 0, currentid, AESKey, false)
			_, err = (*controlConn).Write(respData)
		}
		return
	}

	respData, _ := ConstructPayload(nodeid, "COMMAND", "FILENAME", " ", info.Name(), 0, currentid, AESKey, false)
	_, err = (*controlConn).Write(respData)
	if <-getName {
		getresp = true
		buff := make([]byte, 10240)
		fileHandle, _ := os.Open(filename) //打开文件
		defer fileHandle.Close()           //关闭文件
		if Notagent {
			fmt.Println("\n[*]File transmitting, please wait...")
		}
		for {
			finalnum := strconv.Itoa(slicenum)
			n, err := fileHandle.Read(buff) //读取文件内容
			if err != nil {
				if err == io.EOF {
					respData, _ = ConstructPayload(nodeid, "DATA", "EOF", finalnum, " ", 0, currentid, AESKey, false)
					_, err = (*controlConn).Write(respData)
					return
				} else {
					if Notagent {
						fmt.Println("[*]Cannot read the file")
					}
					respData, _ := ConstructPayload(nodeid, "COMMAND", "CANNOTREAD", " ", filename, 0, currentid, AESKey, false)
					_, err = (*controlConn).Write(respData)
					return
				}
			}
			fileData, err := ConstructPayload(nodeid, "DATA", "FILEDATA", finalnum, string(buff[:n]), 0, currentid, AESKey, false)
			if err != nil {
				fmt.Println(err)
			}
			(*controlConn).Write(fileData)
			slicenum++
		}
	} else {
		fmt.Println("[*]File cannot be uploaded!")
		return
	}

}

func DownloadFile(filename string, conn net.Conn, nodeid uint32, currentid uint32, AESKey []byte) {
	respData, _ := ConstructPayload(nodeid, "COMMAND", "DOWNLOADFILE", " ", filename, 0, currentid, AESKey, false)
	_, err := conn.Write(respData)
	if err != nil {
		return
	}
}

func ReceiveFile(controlConnToAdmin *net.Conn, Eof chan string, FileDataMap *IntStrMap, CannotRead chan bool, UploadFile *os.File, AESKey []byte, Notagent bool, currentid uint32) {
	defer UploadFile.Close()
	if Notagent {
		fmt.Println("\n[*]Downloading file,please wait......")
	}
	for {
		select {
		case st := <-Eof:
			slicetotal, _ := strconv.Atoi(st)
			for {
				time.Sleep(2 * time.Second)
				FileDataMap.RLock()
				if len(FileDataMap.Payload) == slicetotal {
					for num := 0; num < slicetotal; num++ {
						content := FileDataMap.Payload[num]
						_, err := UploadFile.Write([]byte(content))
						if err != nil {
							return
						}
					}
					FileDataMap.RUnlock()
					if Notagent {
						fmt.Println("[*]Transmission complete")
					} else {
						respData, _ := ConstructPayload(0, "COMMAND", "TRANSSUCCESS", " ", " ", 0, currentid, AESKey, false)
						(*controlConnToAdmin).Write(respData)
					}
					return
				}
			}
		case <-CannotRead:
			return
		}
	}
}

func CountDown(getresp *bool, getName chan bool) {
	time.Sleep(10 * time.Second)
	if *getresp == false {
		getName <- false
	}
}
