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

// payload 结构设计时考虑的不是很好，会存在浪费头长度的问题
// 平均差不多是0.02%的浪费

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

	routeLength := make([]byte, config.ROUTE_LEN)
	typeLength := make([]byte, config.TYPE_LEN)
	commandLength := make([]byte, config.COMMAND_LEN)
	fileSliceLength := make([]byte, config.FILESLICENUM_LEN)
	infoLength := make([]byte, config.INFO_LEN)
	Clientid := make([]byte, config.CLIENT_LEN)

	Nodeid := []byte(nodeid)
	routeData := []byte(route)
	ptypeData := []byte(ptype)
	Command := []byte(command)
	fileSliceNumData := []byte(fileSliceNum)
	Info := []byte(info)
	Currentid := []byte(currentid)

	if len(key) != 0 && !pass {
		key, err := crypto.KeyPadding(key)
		if err != nil {
			log.Fatal(err)
		}
		ptypeData = crypto.AESEncrypt(ptypeData, key)
		Command = crypto.AESEncrypt(Command, key)
		Info = crypto.AESEncrypt(Info, key)
	}

	if len(key) != 0 && pass {
		key, err := crypto.KeyPadding(key)
		if err != nil {
			log.Fatal(err)
		}
		ptypeData = crypto.AESEncrypt(ptypeData, key)
		Command = crypto.AESEncrypt(Command, key)
	}

	binary.BigEndian.PutUint32(routeLength, uint32(len(routeData)))
	binary.BigEndian.PutUint32(typeLength, uint32(len(ptypeData)))
	binary.BigEndian.PutUint32(commandLength, uint32(len(Command)))
	binary.BigEndian.PutUint32(fileSliceLength, uint32(len(fileSliceNumData)))
	binary.BigEndian.PutUint32(infoLength, uint32(len(Info)))
	binary.BigEndian.PutUint32(Clientid, clientid)

	buffer.Write(Nodeid)
	buffer.Write(routeLength)
	buffer.Write(routeData)
	buffer.Write(typeLength)
	buffer.Write(ptypeData)
	buffer.Write(commandLength)
	buffer.Write(Command)
	buffer.Write(fileSliceLength)
	buffer.Write(fileSliceNumData)
	buffer.Write(infoLength)
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
		nodeLen         = make([]byte, config.NODE_LEN)
		routeLen        = make([]byte, config.ROUTE_LEN)
		typeLen         = make([]byte, config.TYPE_LEN)
		commandLen      = make([]byte, config.COMMAND_LEN)
		fileSliceNumLen = make([]byte, config.FILESLICENUM_LEN)
		infoLen         = make([]byte, config.INFO_LEN)
		clientidLen     = make([]byte, config.CLIENT_LEN)
		currentidLen    = make([]byte, config.NODE_LEN)
	)

	if len(key) != 0 {
		key, _ = crypto.KeyPadding(key)
	}

	_, err := io.ReadFull(conn, nodeLen)
	if err != nil {
		return payload, err
	}
	payload.NodeId = string(nodeLen)

	_, err = io.ReadFull(conn, routeLen)
	if err != nil {
		return payload, err
	}
	payload.RouteLength = binary.BigEndian.Uint32(routeLen)

	routeBuffer := make([]byte, payload.RouteLength)
	_, err = io.ReadFull(conn, routeBuffer)
	if err != nil {
		return payload, err
	}
	payload.Route = string(routeBuffer)

	_, err = io.ReadFull(conn, typeLen)
	if err != nil {
		return payload, err
	}
	payload.TypeLength = binary.BigEndian.Uint32(typeLen)

	typeBuffer := make([]byte, payload.TypeLength)
	_, err = io.ReadFull(conn, typeBuffer)
	if err != nil {
		return payload, err
	}
	if len(key) != 0 {
		payload.Type = string(crypto.AESDecrypt(typeBuffer[:], key)) //处理lowernodeconn的时候解密type，但是不解密info，防止性能损失
	} else {
		payload.Type = string(typeBuffer[:])
	}

	_, err = io.ReadFull(conn, commandLen)
	if err != nil {
		return payload, err
	}
	payload.CommandLength = binary.BigEndian.Uint32(commandLen)

	commandBuffer := make([]byte, payload.CommandLength)
	_, err = io.ReadFull(conn, commandBuffer)
	if err != nil {
		return payload, err
	}
	if len(key) != 0 {
		payload.Command = string(crypto.AESDecrypt(commandBuffer[:], key))
	} else {
		payload.Command = string(commandBuffer[:])
	}

	_, err = io.ReadFull(conn, fileSliceNumLen)
	if err != nil {
		return payload, err
	}
	payload.FileSliceNumLength = binary.BigEndian.Uint32(fileSliceNumLen)

	fileSliceNumBuffer := make([]byte, payload.FileSliceNumLength)
	_, err = io.ReadFull(conn, fileSliceNumBuffer)
	if err != nil {
		return payload, err
	}
	payload.FileSliceNum = string(fileSliceNumBuffer)

	_, err = io.ReadFull(conn, infoLen)
	if err != nil {
		return payload, err
	}
	payload.InfoLength = binary.BigEndian.Uint32(infoLen)

	infoBuffer := make([]byte, payload.InfoLength)
	_, err = io.ReadFull(conn, infoBuffer)
	if err != nil {
		return payload, err
	}
	if len(key) != 0 && (payload.NodeId == currentid || isinit) {
		payload.Info = string(crypto.AESDecrypt(infoBuffer[:], key))
	} else {
		payload.Info = string(infoBuffer[:])
	}

	_, err = io.ReadFull(conn, clientidLen)
	if err != nil {
		return payload, err
	}
	payload.Clientid = binary.BigEndian.Uint32(clientidLen)

	_, err = io.ReadFull(conn, currentidLen)
	if err != nil {
		return payload, err
	}
	payload.CurrentId = string(currentidLen)

	return payload, nil
}
