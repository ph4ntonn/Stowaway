/*
 * @Author: ph4ntom
 * @Date: 2021-03-09 14:02:57
 * @LastEditors: ph4ntom
 * @LastEditTime: 2021-03-18 19:19:42
 */
package protocol

import (
	"Stowaway/crypto"
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"net"
)

type TCPMessage struct {
	ID           string
	Conn         net.Conn
	CryptoSecret []byte
	Buffer       []byte
	IsPass       bool
}

/**
 * @description: Tcp raw meesage do not need special header
 * @param {*}
 * @return {*}
 */
func (message *TCPMessage) ConstructHeader() {}

/**
 * @description: Construct our own raw tcp data
 * @param {*}
 * @return {*}
 */
func (message *TCPMessage) ConstructData(header Header, mess interface{}) {
	var buffer bytes.Buffer
	var tDataBuf []byte
	// First, construct own header
	messageTypeBuf := make([]byte, 2)
	routeLenBuf := make([]byte, 4)

	binary.BigEndian.PutUint16(messageTypeBuf, header.MessageType)
	binary.BigEndian.PutUint32(routeLenBuf, header.RouteLen)

	// Write header into buffer(except for dataLen)
	buffer.Write([]byte(header.Sender))
	buffer.Write([]byte(header.Accepter))
	buffer.Write(messageTypeBuf)
	buffer.Write(routeLenBuf)
	buffer.Write([]byte(header.Route))

	// Check if message's data is needed to encrypt
	if message.IsPass && message.Buffer != nil {
		dataLenBuf := make([]byte, 8)
		binary.BigEndian.PutUint64(dataLenBuf, uint64(len(message.Buffer)))
		buffer.Write(dataLenBuf)
		buffer.Write(message.Buffer)
		// Remember to set buffer to nil
		message.IsPass = false
		message.Buffer = nil
	} else {
		switch header.MessageType {
		case HI:
			mmess := mess.(HIMess)
			greetingBuf := []byte(mmess.Greeting)
			greetingLenBuf := make([]byte, 2)
			binary.BigEndian.PutUint16(greetingLenBuf, mmess.GreetingLen)
			isAdminBuf := make([]byte, 2)
			binary.BigEndian.PutUint16(isAdminBuf, mmess.IsAdmin)
			// Collect all spilted data, try encrypt them
			tDataBuf = append(greetingLenBuf, greetingBuf...)
			tDataBuf = append(tDataBuf, isAdminBuf...)
			tDataBuf = crypto.AESEncrypt(tDataBuf, message.CryptoSecret)
		case UUID:
			mmess := mess.(UUIDMess)
			uuidLenBuf := make([]byte, 2)
			binary.BigEndian.PutUint16(uuidLenBuf, mmess.UUIDLen)
			uuidBuf := []byte(mmess.UUID)

			tDataBuf = append(uuidLenBuf, uuidBuf...)
			tDataBuf = crypto.AESEncrypt(tDataBuf, message.CryptoSecret)
		case UUIDRET:
			mmess := mess.(UUIDRetMess)
			OKBuf := make([]byte, 2)
			binary.BigEndian.PutUint16(OKBuf, mmess.OK)

			tDataBuf = OKBuf
			tDataBuf = crypto.AESEncrypt(tDataBuf, message.CryptoSecret)
		case MYINFO:
			mmess := mess.(MyInfo)
			usernameLenBuf := make([]byte, 8)
			binary.BigEndian.PutUint64(usernameLenBuf, mmess.UsernameLen)

			usernameBuf := []byte(mmess.Username)

			hostnameLenBuf := make([]byte, 8)
			binary.BigEndian.PutUint64(hostnameLenBuf, mmess.HostnameLen)

			hostnameBuf := []byte(mmess.Hostname)

			tDataBuf = append(usernameLenBuf, usernameBuf...)
			tDataBuf = append(tDataBuf, hostnameLenBuf...)
			tDataBuf = append(tDataBuf, hostnameBuf...)
			tDataBuf = crypto.AESEncrypt(tDataBuf, message.CryptoSecret)
		case MYMEMO:
			mmess := mess.(MyMemo)
			memoLenBuf := make([]byte, 8)
			binary.BigEndian.PutUint64(memoLenBuf, mmess.MemoLen)

			memoBuf := []byte(mmess.Memo)

			tDataBuf = append(memoLenBuf, memoBuf...)
			tDataBuf = crypto.AESEncrypt(tDataBuf, message.CryptoSecret)
		case SHELLREQ:
			mmess := mess.(ShellReq)
			startBuf := make([]byte, 2)
			binary.BigEndian.PutUint16(startBuf, mmess.Start)

			tDataBuf = startBuf
			tDataBuf = crypto.AESEncrypt(tDataBuf, message.CryptoSecret)
		case SHELLRES:
			mmess := mess.(ShellRes)
			OKBuf := make([]byte, 2)
			binary.BigEndian.PutUint16(OKBuf, mmess.OK)

			tDataBuf = OKBuf
			tDataBuf = crypto.AESEncrypt(tDataBuf, message.CryptoSecret)
		case SHELLCOMMAND:
			mmess := mess.(ShellCommand)
			commandLenBuf := make([]byte, 8)
			binary.BigEndian.PutUint64(commandLenBuf, mmess.CommandLen)

			commandBuf := []byte(mmess.Command)

			tDataBuf = append(commandLenBuf, commandBuf...)
			tDataBuf = crypto.AESEncrypt(tDataBuf, message.CryptoSecret)
		case SHELLRESULT:
			mmess := mess.(ShellResult)
			OKBuf := make([]byte, 2)
			binary.BigEndian.PutUint16(OKBuf, mmess.OK)

			resultLenBuf := make([]byte, 8)
			binary.BigEndian.PutUint64(resultLenBuf, mmess.ResultLen)

			resultBuf := []byte(mmess.Result)

			tDataBuf = append(OKBuf, resultLenBuf...)
			tDataBuf = append(tDataBuf, resultBuf...)
			tDataBuf = crypto.AESEncrypt(tDataBuf, message.CryptoSecret)
		case LISTENREQ:
			mmess := mess.(ListenReq)
			addrLenBuf := make([]byte, 8)
			binary.BigEndian.PutUint64(addrLenBuf, mmess.AddrLen)

			addrBuf := []byte(mmess.Addr)

			tDataBuf = append(addrLenBuf, addrBuf...)
			tDataBuf = crypto.AESEncrypt(tDataBuf, message.CryptoSecret)
		case LISTENRES:
			mmess := mess.(ListenRes)
			OKBuf := make([]byte, 2)
			binary.BigEndian.PutUint16(OKBuf, mmess.OK)

			tDataBuf = OKBuf
			tDataBuf = crypto.AESEncrypt(tDataBuf, message.CryptoSecret)
		case SSHREQ:
			mmess := mess.(SSHReq)
			methodBuf := make([]byte, 2)
			binary.BigEndian.PutUint16(methodBuf, mmess.Method)

			usernameLenBuf := make([]byte, 8)
			binary.BigEndian.PutUint64(usernameLenBuf, mmess.UsernameLen)

			usernameBuf := []byte(mmess.Username)

			passwordLenBuf := make([]byte, 8)
			binary.BigEndian.PutUint64(passwordLenBuf, mmess.PasswordLen)

			passwordBuf := []byte(mmess.Password)

			certificateLenBuf := make([]byte, 8)
			binary.BigEndian.PutUint64(certificateLenBuf, mmess.CertificateLen)

			certificateBuf := mmess.Certificate

			tDataBuf = append(methodBuf, usernameLenBuf...)
			tDataBuf = append(tDataBuf, usernameBuf...)
			tDataBuf = append(tDataBuf, passwordLenBuf...)
			tDataBuf = append(tDataBuf, passwordBuf...)
			tDataBuf = append(tDataBuf, certificateLenBuf...)
			tDataBuf = append(tDataBuf, certificateBuf...)
			tDataBuf = crypto.AESEncrypt(tDataBuf, message.CryptoSecret)
		case SSHRES:
			mmess := mess.(SSHRes)
			OKBuf := make([]byte, 2)
			binary.BigEndian.PutUint16(OKBuf, mmess.OK)

			tDataBuf = OKBuf
			tDataBuf = crypto.AESEncrypt(tDataBuf, message.CryptoSecret)
		case SSHCOMMAND:
			mmess := mess.(SSHCommand)

			commandLenBuf := make([]byte, 8)
			binary.BigEndian.PutUint64(commandLenBuf, mmess.CommandLen)

			commandBuf := []byte(mmess.Command)

			tDataBuf = append(commandLenBuf, commandBuf...)
			tDataBuf = crypto.AESEncrypt(tDataBuf, message.CryptoSecret)
		case SSHRESULT:
			mmess := mess.(ShellResult)

			resultLenBuf := make([]byte, 8)
			binary.BigEndian.PutUint64(resultLenBuf, mmess.ResultLen)

			resultBuf := []byte(mmess.Result)

			tDataBuf = append(resultLenBuf, resultBuf...)
			tDataBuf = crypto.AESEncrypt(tDataBuf, message.CryptoSecret)
		default:
		}
	}
	// Calculate the whole data's length
	dataLenBuf := make([]byte, 8)
	binary.BigEndian.PutUint64(dataLenBuf, uint64(len(tDataBuf)))
	buffer.Write(dataLenBuf)
	buffer.Write(tDataBuf)

	message.Buffer = buffer.Bytes()
}

/**
 * @description: Tcp raw meesage do not need special suffix
 * @param {*}
 * @return {*}
 */
func (message *TCPMessage) ConstructSuffix() {}

/**
 * @description: Tcp raw meesage do not need to deconstruct special header
 * @param {*}
 * @return {*}
 */
func (message *TCPMessage) DeconstructHeader() {}

/**
 * @description: Deconstruct our own raw tcp data
 * @param {*}
 * @return {*}
 */
func (message *TCPMessage) DeconstructData() (Header, interface{}, error) {
	var (
		header         = Header{}
		senderBuf      = make([]byte, 10)
		accepterBuf    = make([]byte, 10)
		messageTypeBuf = make([]byte, 2)
		routeLenBuf    = make([]byte, 4)
		dataLenBuf     = make([]byte, 8)
	)

	var err error

	_, err = io.ReadFull(message.Conn, senderBuf)
	if err != nil {
		return header, nil, err
	}
	header.Sender = string(senderBuf)

	_, err = io.ReadFull(message.Conn, accepterBuf)
	if err != nil {
		return header, nil, err
	}
	header.Accepter = string(accepterBuf)

	_, err = io.ReadFull(message.Conn, messageTypeBuf)
	if err != nil {
		return header, nil, err
	}
	header.MessageType = binary.BigEndian.Uint16(messageTypeBuf)

	_, err = io.ReadFull(message.Conn, routeLenBuf)
	if err != nil {
		return header, nil, err
	}
	header.RouteLen = binary.BigEndian.Uint32(routeLenBuf)

	routeBuf := make([]byte, header.RouteLen)
	_, err = io.ReadFull(message.Conn, routeBuf)
	if err != nil {
		return header, nil, err
	}
	header.Route = string(routeBuf)

	_, err = io.ReadFull(message.Conn, dataLenBuf)
	if err != nil {
		return header, nil, err
	}
	header.DataLen = binary.BigEndian.Uint64(dataLenBuf)

	dataBuf := make([]byte, header.DataLen)
	_, err = io.ReadFull(message.Conn, dataBuf)
	if err != nil {
		return header, nil, err
	}

	var fDataBuf []byte
	if header.Accepter == TEMP_UUID || message.ID == ADMIN_UUID || message.ID == header.Accepter {
		fDataBuf = crypto.AESDecrypt(dataBuf[:], message.CryptoSecret)
	} else if message.CryptoSecret == nil {
	} else {
		message.IsPass = true
		message.Buffer = dataBuf
		return header, nil, nil
	}

	switch header.MessageType {
	case HI:
		mmess := new(HIMess)
		mmess.GreetingLen = binary.BigEndian.Uint16(fDataBuf[:2])
		mmess.Greeting = string(fDataBuf[2 : 2+mmess.GreetingLen])
		mmess.IsAdmin = binary.BigEndian.Uint16(fDataBuf[2+mmess.GreetingLen : header.DataLen])
		return header, mmess, nil
	case UUID:
		mmess := new(UUIDMess)
		mmess.UUIDLen = binary.BigEndian.Uint16(fDataBuf[:2])
		mmess.UUID = string(fDataBuf[2 : 2+mmess.UUIDLen])
		return header, mmess, nil
	case UUIDRET:
		mmess := new(UUIDRetMess)
		mmess.OK = binary.BigEndian.Uint16(fDataBuf[:2])
		return header, mmess, nil
	case MYINFO:
		mmess := new(MyInfo)
		mmess.UsernameLen = binary.BigEndian.Uint64(fDataBuf[:8])
		mmess.Username = string(fDataBuf[8 : 8+mmess.UsernameLen])
		mmess.HostnameLen = binary.BigEndian.Uint64(fDataBuf[8+mmess.UsernameLen : 16+mmess.UsernameLen])
		mmess.Hostname = string(fDataBuf[16+mmess.UsernameLen : 16+mmess.UsernameLen+mmess.HostnameLen])
		return header, mmess, nil
	case MYMEMO:
		mmess := new(MyMemo)
		mmess.MemoLen = binary.BigEndian.Uint64(fDataBuf[:8])
		mmess.Memo = string(fDataBuf[8 : 8+mmess.MemoLen])
		return header, mmess, nil
	case SHELLREQ:
		mmess := new(ShellReq)
		mmess.Start = binary.BigEndian.Uint16(fDataBuf[:2])
		return header, mmess, nil
	case SHELLRES:
		mmess := new(ShellRes)
		mmess.OK = binary.BigEndian.Uint16(fDataBuf[:2])
		return header, mmess, nil
	case SHELLCOMMAND:
		mmess := new(ShellCommand)
		mmess.CommandLen = binary.BigEndian.Uint64(fDataBuf[:8])
		mmess.Command = string(fDataBuf[8 : 8+mmess.CommandLen])
		return header, mmess, nil
	case SHELLRESULT:
		mmess := new(ShellResult)
		mmess.OK = binary.BigEndian.Uint16(fDataBuf[:2])
		mmess.ResultLen = binary.BigEndian.Uint64(fDataBuf[2:10])
		mmess.Result = string(fDataBuf[10 : 10+mmess.ResultLen])
		return header, mmess, nil
	case LISTENREQ:
		mmess := new(ListenReq)
		mmess.AddrLen = binary.BigEndian.Uint64(fDataBuf[:8])
		mmess.Addr = string(fDataBuf[8 : 8+mmess.AddrLen])
		return header, mmess, nil
	case LISTENRES:
		mmess := new(ListenRes)
		mmess.OK = binary.BigEndian.Uint16(fDataBuf[:2])
		return header, mmess, nil
	case SSHREQ:
		mmess := new(SSHReq)
		mmess.Method = binary.BigEndian.Uint16(fDataBuf[:2])
		mmess.UsernameLen = binary.BigEndian.Uint64(fDataBuf[2:10])
		mmess.Username = string(fDataBuf[10 : 10+mmess.UsernameLen])
		mmess.PasswordLen = binary.BigEndian.Uint64(fDataBuf[10+mmess.UsernameLen : 18+mmess.UsernameLen])
		mmess.Password = string(fDataBuf[18+mmess.UsernameLen : 18+mmess.UsernameLen+mmess.PasswordLen])
		mmess.CertificateLen = binary.BigEndian.Uint64(fDataBuf[18+mmess.UsernameLen+mmess.PasswordLen : 26+mmess.UsernameLen+mmess.PasswordLen])
		mmess.Certificate = fDataBuf[26+mmess.UsernameLen+mmess.PasswordLen : 26+mmess.UsernameLen+mmess.PasswordLen+mmess.CertificateLen]
		return header, mmess, nil
	case SSHRES:
		mmess := new(SSHRes)
		mmess.OK = binary.BigEndian.Uint16(fDataBuf[:2])
		return header, mmess, nil
	case SSHCOMMAND:
		mmess := new(SSHCommand)
		mmess.CommandLen = binary.BigEndian.Uint64(fDataBuf[:8])
		mmess.Command = string(fDataBuf[8 : 8+mmess.CommandLen])
		return header, mmess, nil
	case SSHRESULT:
		mmess := new(SSHResult)
		mmess.ResultLen = binary.BigEndian.Uint64(fDataBuf[:8])
		mmess.Result = string(fDataBuf[8 : 8+mmess.ResultLen])
		return header, mmess, nil
	default:
	}

	return header, nil, errors.New("Unknown error!")
}

/**
 * @description: Tcp raw meesage do not need to deconstruct special suffix
 * @param {*}
 * @return {*}
 */
func (message *TCPMessage) DeconstructSuffix() {}

/**
 * @description: Send message to peer node
 * @param {*}
 * @return {*}
 */
func (message *TCPMessage) SendMessage() {
	message.Conn.Write(message.Buffer)
	message.Buffer = nil
}
