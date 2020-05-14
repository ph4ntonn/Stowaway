package utils

import (
	"bytes"
	"encoding/binary"
	"io"
	"log"
	"net"

	"Stowaway/config"
	"Stowaway/crypto"
)

type Payload struct {
	NodeId string //接收节点序号

	RouteLength uint32 //路由长度

	Route string //路由表

	TypeLength uint32 //标识符长度

	Type string //标示是data还是command

	CommandLength uint32 //命令长度

	Command string //命令类型

	FileSliceNumLength uint32 //文件传输分包序号字段长度

	FileSliceNum string //文件传输分包序号

	InfoLength uint32 //载荷长度

	Info string //具体载荷

	Clientid uint32 //socks以及forward功能中用来标识当前需要操作的connection

	CurrentId string //当前节点序号
}

// ConstructPayload 生成并返回payload
func ConstructPayload(nodeid string, route string, ptype string, command string, fileSliceNum string, info string, clientid uint32, currentid string, key []byte, pass bool) ([]byte, error) {
	var buffer bytes.Buffer

	Routelength := make([]byte, config.ROUTE_LEN)
	TypeLength := make([]byte, config.TYPE_LEN)
	CommandLength := make([]byte, config.COMMAND_LEN)
	FilesliceLength := make([]byte, config.FILESLICENUM_LEN)
	InfoLength := make([]byte, config.INFO_LEN)
	Clientid := make([]byte, config.CLIENT_LEN)

	Nodeid := []byte(nodeid)
	Routedata := []byte(route)
	PtypeData := []byte(ptype)
	Command := []byte(command)
	FileSliceNumData := []byte(fileSliceNum)
	Info := []byte(info)
	Currentid := []byte(currentid)

	if len(key) != 0 && !pass {
		key, err := crypto.KeyPadding(key)
		if err != nil {
			log.Fatal(err)
		}
		PtypeData = crypto.AESEncrypt(PtypeData, key)
		Command = crypto.AESEncrypt(Command, key)
		Info = crypto.AESEncrypt(Info, key)
	}

	if len(key) != 0 && pass {
		key, err := crypto.KeyPadding(key)
		if err != nil {
			log.Fatal(err)
		}
		PtypeData = crypto.AESEncrypt(PtypeData, key)
		Command = crypto.AESEncrypt(Command, key)
	}

	binary.BigEndian.PutUint32(Routelength, uint32(len(Routedata)))
	binary.BigEndian.PutUint32(TypeLength, uint32(len(PtypeData)))
	binary.BigEndian.PutUint32(CommandLength, uint32(len(Command)))
	binary.BigEndian.PutUint32(FilesliceLength, uint32(len(FileSliceNumData)))
	binary.BigEndian.PutUint32(InfoLength, uint32(len(Info)))
	binary.BigEndian.PutUint32(Clientid, clientid)

	buffer.Write(Nodeid)
	buffer.Write(Routelength)
	buffer.Write(Routedata)
	buffer.Write(TypeLength)
	buffer.Write(PtypeData)
	buffer.Write(CommandLength)
	buffer.Write(Command)
	buffer.Write(FilesliceLength)
	buffer.Write(FileSliceNumData)
	buffer.Write(InfoLength)
	buffer.Write(Info)
	buffer.Write(Clientid)
	buffer.Write(Currentid)

	payload := buffer.Bytes()

	return payload, nil
}

// ExtractPayload 解析并返回payload
func ExtractPayload(conn net.Conn, key []byte, currentid string, isinit bool) (*Payload, error) {
	var (
		payload         = &Payload{}
		nodelen         = make([]byte, config.NODE_LEN)
		routelen        = make([]byte, config.ROUTE_LEN)
		typelen         = make([]byte, config.TYPE_LEN)
		commandlen      = make([]byte, config.COMMAND_LEN)
		fileslicenumlen = make([]byte, config.FILESLICENUM_LEN)
		infolen         = make([]byte, config.INFO_LEN)
		clientidlen     = make([]byte, config.CLIENT_LEN)
		currentidlen    = make([]byte, config.NODE_LEN)
	)

	if len(key) != 0 {
		key, _ = crypto.KeyPadding(key)
	}

	_, err := io.ReadFull(conn, nodelen)
	if err != nil {
		return payload, err
	}
	payload.NodeId = string(nodelen)

	_, err = io.ReadFull(conn, routelen)
	if err != nil {
		return payload, err
	}
	payload.RouteLength = binary.BigEndian.Uint32(routelen)

	routebuffer := make([]byte, payload.RouteLength)
	_, err = io.ReadFull(conn, routebuffer)
	if err != nil {
		return payload, err
	}
	payload.Route = string(routebuffer)

	_, err = io.ReadFull(conn, typelen)
	if err != nil {
		return payload, err
	}
	payload.TypeLength = binary.BigEndian.Uint32(typelen)

	typebuffer := make([]byte, payload.TypeLength)
	_, err = io.ReadFull(conn, typebuffer)
	if err != nil {
		return payload, err
	}
	if len(key) != 0 {
		payload.Type = string(crypto.AESDecrypt(typebuffer[:], key)) //处理lowernodeconn的时候解密type，但是不解密info，防止性能损失
	} else {
		payload.Type = string(typebuffer[:])
	}

	_, err = io.ReadFull(conn, commandlen)
	if err != nil {
		return payload, err
	}
	payload.CommandLength = binary.BigEndian.Uint32(commandlen)

	commandbuffer := make([]byte, payload.CommandLength)
	_, err = io.ReadFull(conn, commandbuffer)
	if err != nil {
		return payload, err
	}
	if len(key) != 0 {
		payload.Command = string(crypto.AESDecrypt(commandbuffer[:], key))
	} else {
		payload.Command = string(commandbuffer[:])
	}

	_, err = io.ReadFull(conn, fileslicenumlen)
	if err != nil {
		return payload, err
	}
	payload.FileSliceNumLength = binary.BigEndian.Uint32(fileslicenumlen)

	fileslicenumbuffer := make([]byte, payload.FileSliceNumLength)
	_, err = io.ReadFull(conn, fileslicenumbuffer)
	if err != nil {
		return payload, err
	}
	payload.FileSliceNum = string(fileslicenumbuffer)

	_, err = io.ReadFull(conn, infolen)
	if err != nil {
		return payload, err
	}
	payload.InfoLength = binary.BigEndian.Uint32(infolen)

	infobuffer := make([]byte, payload.InfoLength)
	_, err = io.ReadFull(conn, infobuffer)
	if err != nil {
		return payload, err
	}
	if len(key) != 0 && (payload.NodeId == currentid || isinit) {
		payload.Info = string(crypto.AESDecrypt(infobuffer[:], key))
	} else {
		payload.Info = string(infobuffer[:])
	}

	_, err = io.ReadFull(conn, clientidlen)
	if err != nil {
		return payload, err
	}
	payload.Clientid = binary.BigEndian.Uint32(clientidlen)

	_, err = io.ReadFull(conn, currentidlen)
	if err != nil {
		return payload, err
	}
	payload.CurrentId = string(currentidlen)

	return payload, nil
}
