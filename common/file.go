package common

import (
	"fmt"
	"math"
	"net"
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/cheggaaa/pb/v3"
)

var File *FileStatus
var Bar *pb.ProgressBar

func init() {
	File = NewFileStatus()
}

/*-------------------------上传/下载文件相关代码--------------------------*/
//admin || agent上传文件
func UploadFile(route, filename string, controlConn *net.Conn, nodeid string, getName chan bool, AESKey []byte, currentid string, Notagent bool) {
	var slicenum int = 0

	info, err := os.Stat(filename)
	if err != nil {
		if Notagent {
			fmt.Println("[*]File not found!")
		} else {
			respData, _ := ConstructPayload(nodeid, route, "COMMAND", "FILENOTEXIST", " ", filename, 0, currentid, AESKey, false) //发送文件是否存在的情况
			(*controlConn).Write(respData)
		}
		return
	}

	respData, _ := ConstructPayload(nodeid, route, "COMMAND", "FILENAME", " ", info.Name(), 0, currentid, AESKey, false) //发送文件名
	(*controlConn).Write(respData)

	if <-getName {
		buff := make([]byte, 10240)
		fileHandle, _ := os.Open(filename) //打开文件
		defer fileHandle.Close()           //关闭文件

		fileInfo, _ := fileHandle.Stat()
		if fileInfo == nil {
			if Notagent {
				fmt.Println("[*]Cannot read the file")
			}
			respData, _ := ConstructPayload(nodeid, route, "COMMAND", "CANNOTREAD", " ", info.Name(), 0, currentid, AESKey, false) //检查是否能读
			(*controlConn).Write(respData)
			return
		}

		fileSliceNum := math.Ceil(float64(fileInfo.Size()) / 10240)
		fileSliceStr := strconv.FormatInt(int64(fileSliceNum), 10) //计算文件需要被分多少包

		respData, _ = ConstructPayload(nodeid, route, "COMMAND", "FILESLICENUM", " ", fileSliceStr, 0, currentid, AESKey, false)
		(*controlConn).Write(respData) //告知包数量

		if Notagent {
			fmt.Println("\n[*]File transmitting, please wait...")
			Bar = NewBar(fileInfo.Size())
			Bar.Start()
		}

		<-File.TotalConfirm //当对端确定接收到包数量通知后继续

		filesize := strconv.FormatInt(fileInfo.Size(), 10)
		respData, _ = ConstructPayload(nodeid, route, "COMMAND", "FILESIZE", " ", filesize, 0, currentid, AESKey, false)
		(*controlConn).Write(respData) //告知文件大小

		<-File.TotalConfirm //当对端确定接收到文件大小通知后继续

		for {
			n, err := fileHandle.Read(buff) //读取文件内容
			if err != nil {
				if Notagent {
					Bar.Finish()
				}
				return
			}
			fileData, _ := ConstructPayload(nodeid, route, "DATA", "FILEDATA", strconv.Itoa(slicenum), string(buff[:n]), 0, currentid, AESKey, false) //发送文件内容
			(*controlConn).Write(fileData)
			slicenum++
			if Notagent {
				Bar.Add64(int64(n))
			}
		}
	} else {
		if !Notagent {
			respData, _ = ConstructPayload(AdminId, route, "COMMAND", "CANNOTUPLOAD", " ", info.Name(), 0, currentid, AESKey, false) //对端没有拿到文件名
			(*controlConn).Write(respData)
		} else {
			fmt.Println("[*]File cannot be uploaded!")
		}
		return
	}
}

//admin下载文件
func DownloadFile(route, filename string, conn net.Conn, nodeid string, currentid string, AESKey []byte) {
	respData, _ := ConstructPayload(nodeid, route, "COMMAND", "DOWNLOADFILE", " ", filename, 0, currentid, AESKey, false)
	_, err := conn.Write(respData)
	if err != nil {
		return
	}
}

//admin || agent接收文件
func ReceiveFile(route string, controlConnToAdmin *net.Conn, FileDataMap *IntStrMap, CannotRead chan bool, UploadFile *os.File, AESKey []byte, Notagent bool, currentid string) {
	defer UploadFile.Close()
	if Notagent {
		fmt.Println("\n[*]Downloading file,please wait......")
	}

	<-File.ReceiveFileSliceNum //确认收到分包数量
	<-File.ReceiveFileSize     //确认收到文件大小

	if Notagent {
		Bar = NewBar(File.FileSize)
		Bar.Start()
	}

	for num := 0; num < File.TotalSilceNum; num++ { //根据对端传输过来的文件分包数进行循环
		for {
			if len(CannotRead) != 1 { //检查不可读chan是否存在元素，存在即表示对端无法继续读，上传/下载终止
				FileDataMap.Lock()
				if _, ok := FileDataMap.Payload[num]; ok {
					if Notagent {
						Bar.Add64(int64(len(FileDataMap.Payload[num])))
					}
					UploadFile.Write([]byte(FileDataMap.Payload[num]))
					delete(FileDataMap.Payload, num) //往文件里写完后立即清空，防止占用内存过大
					FileDataMap.Unlock()
					break
				} else {
					FileDataMap.Unlock()
					time.Sleep(5 * time.Millisecond) //如果暂时没有收到当前序号的包，先释放锁，等待5ms后继续检查
				}
			} else {
				<-CannotRead
				return
			}
		}
	}

	runtime.GC() //进行一次gc

	if Notagent {
		Bar.Finish()
		fmt.Println("[*]Transmission complete")
	} else {
		respData, _ := ConstructPayload(AdminId, route, "COMMAND", "TRANSSUCCESS", " ", " ", 0, currentid, AESKey, false)
		(*controlConnToAdmin).Write(respData)
	}
	return
}

//进度条
func NewBar(length int64) *pb.ProgressBar {
	bar := pb.New64(int64(length))
	bar.SetTemplate(pb.Full)
	bar.Set(pb.Bytes, true)
	return bar
}
