package common

import (
	"fmt"
	"io"
	"net"
	"os"
	"time"
)

func UploadFile(filename string, ControlConn net.Conn, DataConn net.Conn, nodeid uint32, GetName *chan bool, AESKey []byte, Notagent bool) {
	var getresp bool = false
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
	if <-*GetName {
		getresp = true
		buff := make([]byte, 10240)
		fileHandle, _ := os.Open(filename) //打开文件
		defer fileHandle.Close()           //关闭文件
		for {
			n, err := fileHandle.Read(buff) //读取文件内容
			if err != nil {
				if err == io.EOF {
					if Notagent {
						fmt.Printf("\nFile %s transmission complete!\n", info.Name())
					}
					respData, _ = ConstructDataResult(nodeid, 0, " ", "EOF", " ", AESKey, 0)
					_, err = DataConn.Write(respData)
					return
				} else {
					if Notagent {
						fmt.Println("Cannot read the file")
					} else {
						respData, _ := ConstructCommand("CANNOTREAD", filename, nodeid, AESKey)
						_, err = ControlConn.Write(respData)
					}
					return
				}
			}
			fileData, _ := ConstructDataResult(nodeid, 0, " ", "FILEDATA", string(buff[:n]), AESKey, 0)
			DataConn.Write(fileData)
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

func ReceiveFile(Eof *chan bool, FileData *chan string, UploadFile *os.File) {
	defer UploadFile.Close()
	for {
		select {
		case <-*Eof:
			fmt.Println("Transmission complete")
			return
		default:
			content := <-*FileData
			_, err := UploadFile.Write([]byte(content))
			if err != nil {
				return
			}
		}
	}
}

func CountDown(getresp *bool, GetName *chan bool) {
	time.Sleep(10 * time.Second)
	if *getresp == false {
		*GetName <- false
	}
}
