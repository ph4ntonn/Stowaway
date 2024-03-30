package share

import (
	"fmt"
	"io"
	"os"
	"runtime"

	"Stowaway/global"
	"Stowaway/protocol"
)

const (
	ADMIN = iota
	AGENT
	// status
	START
	ADD
	DONE
)

type MyFile struct {
	FileName   string
	FilePath   string
	FileSize   int64
	SliceSize  int64
	SliceNum   uint64
	ErrChan    chan bool
	DataChan   chan []byte
	StatusChan chan *Status
	Handler    *os.File
}

type Status struct {
	Stat  int
	Scale int64
}

func NewFile() *MyFile {
	file := new(MyFile)
	file.SliceSize = 30720
	file.ErrChan = make(chan bool)
	file.DataChan = make(chan []byte)
	file.StatusChan = make(chan *Status, 10) // Give buffer,make sure file transmitting won't be blocked when passing Status to admin
	return file
}

func (file *MyFile) SendFileStat(route string, targetUUID string, identity int) error {
	var err error
	var sMessage protocol.Message
	if identity == ADMIN {
		sMessage = protocol.NewDownMsg(global.G_Component.Conn, global.G_Component.Secret, global.G_Component.UUID)
	} else {
		sMessage = protocol.NewUpMsg(global.G_Component.Conn, global.G_Component.Secret, global.G_Component.UUID)
	}

	statHeader := &protocol.Header{
		Sender:      global.G_Component.UUID,
		Accepter:    targetUUID,
		MessageType: protocol.FILESTATREQ,
		RouteLen:    uint32(len([]byte(route))),
		Route:       route,
	}

	downHeader := &protocol.Header{
		Sender:      global.G_Component.UUID,
		Accepter:    targetUUID,
		MessageType: protocol.FILEDOWNRES,
		RouteLen:    uint32(len([]byte(route))),
		Route:       route,
	}

	defer func() {
		if err != nil && identity == AGENT {
			fileDownResMess := &protocol.FileDownRes{
				OK: 0,
			}
			protocol.ConstructMessage(sMessage, downHeader, fileDownResMess, false)
			sMessage.SendMessage()
		}
	}()

	fileHandler, err := os.Open(file.FilePath)
	if err != nil {
		return err
	}
	file.Handler = fileHandler

	fileInfo, err := fileHandler.Stat()
	if err != nil {
		fileHandler.Close()
		return err
	}

	file.FileSize = fileInfo.Size()
	fileSliceNum := file.FileSize / file.SliceSize
	remain := file.FileSize % file.SliceSize
	if remain != 0 {
		fileSliceNum++
	}

	fileStatReqMess := &protocol.FileStatReq{
		FilenameLen: uint32(len([]byte(file.FileName))),
		Filename:    file.FileName,
		FileSize:    uint64(file.FileSize),
		SliceNum:    uint64(fileSliceNum),
	}

	protocol.ConstructMessage(sMessage, statHeader, fileStatReqMess, false)
	sMessage.SendMessage()

	return nil
}

func (file *MyFile) CheckFileStat(route string, targetUUID string, identity int) error {
	var err error
	var sMessage protocol.Message

	if identity == ADMIN {
		sMessage = protocol.NewDownMsg(global.G_Component.Conn, global.G_Component.Secret, global.G_Component.UUID)
	} else {
		sMessage = protocol.NewUpMsg(global.G_Component.Conn, global.G_Component.Secret, global.G_Component.UUID)
	}

	header := &protocol.Header{
		Sender:      global.G_Component.UUID,
		Accepter:    targetUUID,
		MessageType: protocol.FILESTATRES,
		RouteLen:    uint32(len([]byte(route))),
		Route:       route,
	}

	fileStatResSuccMess := &protocol.FileStatRes{
		OK: 1,
	}

	fileStatResFailMess := &protocol.FileStatRes{
		OK: 0,
	}

	defer func() {
		if err != nil {
			protocol.ConstructMessage(sMessage, header, fileStatResFailMess, false)
			sMessage.SendMessage()
		}
	}()

	fileHandler, err := os.Create(file.FileName)
	if err != nil {
		return err
	}

	file.Handler = fileHandler

	protocol.ConstructMessage(sMessage, header, fileStatResSuccMess, false)
	sMessage.SendMessage()

	return nil
}

func (file *MyFile) Upload(route string, targetUUID string, identity int) {
	var sMessage protocol.Message
	if identity == ADMIN {
		sMessage = protocol.NewDownMsg(global.G_Component.Conn, global.G_Component.Secret, global.G_Component.UUID)
	} else {
		sMessage = protocol.NewUpMsg(global.G_Component.Conn, global.G_Component.Secret, global.G_Component.UUID)
	}

	dataHeader := &protocol.Header{
		Sender:      global.G_Component.UUID,
		Accepter:    targetUUID,
		MessageType: protocol.FILEDATA,
		RouteLen:    uint32(len([]byte(route))),
		Route:       route,
	}

	errHeader := &protocol.Header{
		Sender:      global.G_Component.UUID,
		Accepter:    targetUUID,
		MessageType: protocol.FILEERR,
		RouteLen:    uint32(len([]byte(route))),
		Route:       route,
	}

	fileErrMess := &protocol.FileErr{
		Error: 1,
	}

	if identity == ADMIN {
		fmt.Println("\n[*] File transmitting, please wait...")
		file.StatusChan <- &Status{Stat: START}
	}

	buffer := make([]byte, 30720)

	defer func() {
		if identity == ADMIN {
			file.StatusChan <- &Status{Stat: DONE}
		}
		runtime.GC()
		file.Handler.Close()
	}()

	for {
		length, err := file.Handler.Read(buffer)
		if err != nil && err != io.EOF {
			protocol.ConstructMessage(sMessage, errHeader, fileErrMess, false)
			sMessage.SendMessage()
			return
		} else if err != nil && err == io.EOF {
			return
		}

		fileDataMess := &protocol.FileData{
			DataLen: uint64(length),
			Data:    buffer[:length],
		}

		protocol.ConstructMessage(sMessage, dataHeader, fileDataMess, false)
		sMessage.SendMessage()

		if identity == ADMIN {
			file.StatusChan <- &Status{Stat: ADD, Scale: int64(length)}
		}
	}

}

func (file *MyFile) Receive(route string, targetUUID string, identity int) {
	if identity == ADMIN {
		fmt.Println("\n[*] File transmitting, please wait...")
		file.StatusChan <- &Status{Stat: START}
	}

	defer func() {
		if identity == ADMIN {
			file.StatusChan <- &Status{Stat: DONE}
		}
		runtime.GC()
		file.Handler.Close()
	}()

	for num := 0; num < int(file.SliceNum); num++ {
		select {
		case <-file.ErrChan:
			return
		case data := <-file.DataChan:
			if identity == ADMIN {
				file.StatusChan <- &Status{Stat: ADD, Scale: int64(len(data))}
			}
			file.Handler.Write(data)
		}
	}
}

func (file *MyFile) Ask4Download(route string, targetUUID string) {
	sMessage := protocol.NewDownMsg(global.G_Component.Conn, global.G_Component.Secret, global.G_Component.UUID)

	header := &protocol.Header{
		Sender:      global.G_Component.UUID,
		Accepter:    targetUUID,
		MessageType: protocol.FILEDOWNREQ,
		RouteLen:    uint32(len([]byte(route))),
		Route:       route,
	}

	fileDownReqMess := &protocol.FileDownReq{
		FilePathLen: uint32(len([]byte(file.FilePath))),
		FilePath:    file.FilePath,
		FilenameLen: uint32(len([]byte(file.FileName))),
		Filename:    file.FileName,
	}

	protocol.ConstructMessage(sMessage, header, fileDownReqMess, false)
	sMessage.SendMessage()
}
