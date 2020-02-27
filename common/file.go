package common

import (
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"sync"
	"time"
)

type SafeFileDataMap struct {
	sync.RWMutex
	FileDataChan map[int]string
}

/*-------------------------上传/下载文件相关代码--------------------------*/
func UploadFile(filename string, ControlConn net.Conn, DataConn net.Conn, nodeid uint32, GetName chan bool, AESKey []byte, currentid uint32, Notagent bool) {
	var getresp bool = false
	var slicenum int = 0
	go CountDown(&getresp, GetName)

	info, err := os.Stat(filename)
	if err != nil {
		getresp = true
		if Notagent {
			fmt.Println("File not found!")
		} else {
			respData, _ := ConstructCommand("FILENOTEXIST", filename, nodeid, AESKey)
			_, err = ControlConn.Write(respData)
		}
		return
	}

	respData, _ := ConstructCommand("FILENAME", info.Name(), nodeid, AESKey)
	_, err = ControlConn.Write(respData)
	if <-GetName {
		getresp = true
		buff := make([]byte, 10240)
		fileHandle, _ := os.Open(filename) //打开文件
		defer fileHandle.Close()           //关闭文件
		if Notagent {
			fmt.Println("\nFile transmitting, please wait...")
		}
		for {
			finalnum := strconv.Itoa(slicenum)
			n, err := fileHandle.Read(buff) //读取文件内容
			if err != nil {
				if err == io.EOF {
					respData, _ = ConstructDataResult(nodeid, 0, finalnum, "EOF", " ", AESKey, currentid)
					_, err = DataConn.Write(respData)
					return
				} else {
					if Notagent {
						fmt.Println("Cannot read the file")
					}
					respData, _ := ConstructCommand("CANNOTREAD", filename, nodeid, AESKey)
					_, err = ControlConn.Write(respData)
					return
				}
			}
			fileData, err := ConstructDataResult(nodeid, 0, finalnum, "FILEDATA", string(buff[:n]), AESKey, currentid)
			if err != nil {
				fmt.Println(err)
			}
			DataConn.Write(fileData)
			slicenum++
		}
	} else {
		fmt.Println("File cannot be uploaded!")
		return
	}

}

func DownloadFile(filename string, conn net.Conn, nodeid uint32, AESKey []byte) {
	respData, _ := ConstructCommand("DOWNLOADFILE", filename, nodeid, AESKey)
	_, err := conn.Write(respData)
	if err != nil {
		return
	}
}

func ReceiveFile(controlConnToAdmin *net.Conn, Eof chan string, FileDataMap *SafeFileDataMap, CannotRead chan bool, UploadFile *os.File, AESKey []byte, Notagent bool) {
	defer UploadFile.Close()
	if Notagent {
		fmt.Println("\nDownloading file,please wait......")
	}
	for {
		select {
		case st := <-Eof:
			slicetotal, _ := strconv.Atoi(st)
			for {
				time.Sleep(2 * time.Second)
				FileDataMap.RLock()
				if len(FileDataMap.FileDataChan) == slicetotal {
					for num := 0; num < slicetotal; num++ {
						content := FileDataMap.FileDataChan[num]
						_, err := UploadFile.Write([]byte(content))
						if err != nil {
							return
						}
					}
					FileDataMap.RUnlock()
					if Notagent {
						fmt.Println("Transmission complete")
					} else {
						respData, _ := ConstructCommand("TRANSSUCCESS", " ", 0, AESKey)
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

func CountDown(getresp *bool, GetName chan bool) {
	time.Sleep(10 * time.Second)
	if *getresp == false {
		GetName <- false
	}
}
