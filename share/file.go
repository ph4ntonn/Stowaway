package share

import (
	"fmt"
	"math"
	"net"
	"os"
	"runtime"
	"strconv"
	"time"

	"Stowaway/utils"

	"github.com/cheggaaa/pb/v3"
)

var File *utils.FileStatus
var Bar *pb.ProgressBar

func init() {
	File = utils.NewFileStatus()
}

/*-------------------------上传/下载文件相关代码--------------------------*/

// UploadFile admin || agent上传文件
func UploadFile(route, filename string, controlConn *net.Conn, nodeid string, getName chan bool, AESKey []byte, currentid string, notAgent bool) {
	var slicenum int = 0

	info, err := os.Stat(filename)
	if err != nil {
		if notAgent {
			fmt.Println("[*]File not found!")
		} else {
			utils.ConstructPayloadAndSend(*controlConn, nodeid, route, "COMMAND", "FILENOTEXIST", " ", filename, 0, currentid, AESKey, false) //发送文件是否存在的情况
		}
		return
	}

	utils.ConstructPayloadAndSend(*controlConn, nodeid, route, "COMMAND", "FILENAME", " ", info.Name(), 0, currentid, AESKey, false) //发送文件名

	if <-getName {
		buff := make([]byte, 30720)
		fileHandle, _ := os.Open(filename) //打开文件
		defer fileHandle.Close()           //关闭文件

		fileInfo, _ := fileHandle.Stat()
		if fileInfo == nil {
			if notAgent {
				fmt.Println("[*]Cannot read the file")
			}
			utils.ConstructPayloadAndSend(*controlConn, nodeid, route, "COMMAND", "CANNOTREAD", " ", info.Name(), 0, currentid, AESKey, false) //检查是否能读
			return
		}

		fileSliceNum := math.Ceil(float64(fileInfo.Size()) / 30720)
		fileSliceStr := strconv.FormatInt(int64(fileSliceNum), 10) //计算文件需要被分多少包

		utils.ConstructPayloadAndSend(*controlConn, nodeid, route, "COMMAND", "FILESLICENUM", " ", fileSliceStr, 0, currentid, AESKey, false) //告知包数量

		if notAgent {
			fmt.Println("\n[*]File transmitting, please wait...")
			Bar = utils.NewBar(fileInfo.Size())
			Bar.Start()
		}

		<-File.TotalConfirm //当对端确定接收到包数量通知后继续

		filesize := strconv.FormatInt(fileInfo.Size(), 10)
		utils.ConstructPayloadAndSend(*controlConn, nodeid, route, "COMMAND", "FILESIZE", " ", filesize, 0, currentid, AESKey, false) //告知文件大小

		<-File.TotalConfirm //当对端确定接收到文件大小通知后继续

		for {
			n, err := fileHandle.Read(buff) //读取文件内容
			if err != nil {
				if notAgent {
					Bar.Finish()
				}
				return
			}

			utils.ConstructPayloadAndSend(*controlConn, nodeid, route, "DATA", "FILEDATA", strconv.Itoa(slicenum), string(buff[:n]), 0, currentid, AESKey, false)
			//文件封包id加一
			slicenum++

			if notAgent {
				Bar.Add64(int64(n))
			}
		}
	} else {
		if !notAgent {
			utils.ConstructPayloadAndSend(*controlConn, utils.AdminId, route, "COMMAND", "CANNOTUPLOAD", " ", info.Name(), 0, currentid, AESKey, false)
		} else {
			fmt.Println("[*]File cannot be uploaded!")
		}
		return
	}
}

// DownloadFile admin下载文件
func DownloadFile(route, fileName string, conn net.Conn, nodeid string, currentid string, AESKey []byte) {
	err := utils.ConstructPayloadAndSend(conn, nodeid, route, "COMMAND", "DOWNLOADFILE", " ", fileName, 0, currentid, AESKey, false)
	if err != nil {
		return
	}
}

// ReceiveFile admin || agent接收文件
func ReceiveFile(route string, controlConnToAdmin *net.Conn, fileDataMap *utils.IntStrMap, cannotRead chan bool, uploadFile *os.File, AESKey []byte, notAgent bool, currentid string) {
	defer uploadFile.Close()

	if notAgent {
		fmt.Println("\n[*]Downloading file,please wait......")
	}

	<-File.ReceiveFileSliceNum //确认收到分包数量
	<-File.ReceiveFileSize     //确认收到文件大小

	if notAgent {
		Bar = utils.NewBar(File.FileSize)
		Bar.Start()
	}

	for num := 0; num < File.TotalSilceNum; num++ { //根据对端传输过来的文件分包数进行循环
		for {
			if len(cannotRead) != 1 { //检查不可读chan是否存在元素，存在即表示对端无法继续读，上传/下载终止
				fileDataMap.Lock()
				if _, ok := fileDataMap.Payload[num]; ok {

					if notAgent {
						Bar.Add64(int64(len(fileDataMap.Payload[num])))
					}

					uploadFile.Write([]byte(fileDataMap.Payload[num]))
					delete(fileDataMap.Payload, num) //往文件里写完后立即清空，防止占用内存过大
					fileDataMap.Unlock()
					break
				} else {
					fileDataMap.Unlock()
					time.Sleep(5 * time.Millisecond) //如果暂时没有收到当前序号的包，先释放锁，等待5ms后继续检查(减少在网络传输过慢时cpu消耗)
				}
			} else {
				<-cannotRead
				return
			}
		}
	}

	if notAgent {
		Bar.Finish()
		fmt.Println("[*]Transmission complete")
	} else {
		utils.ConstructPayloadAndSend(*controlConnToAdmin, utils.AdminId, route, "COMMAND", "TRANSSUCCESS", " ", " ", 0, currentid, AESKey, false)
	}

	runtime.GC() //进行一次gc

	return
}
