package protocol

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"net"

	"Stowaway/crypto"
)

type RawMessage struct {
	// Essential component to apply a Message
	UUID         string
	Conn         net.Conn
	CryptoSecret []byte
	// Prepared buffer
	HeaderBuffer []byte
	DataBuffer   []byte
}

func (message *RawMessage) ConstructHeader() {}

func (message *RawMessage) ConstructData(header *Header, mess interface{}, isPass bool) {
	var headerBuffer, dataBuffer bytes.Buffer
	// First, construct own header
	messageTypeBuf := make([]byte, 2)
	routeLenBuf := make([]byte, 4)

	binary.BigEndian.PutUint16(messageTypeBuf, header.MessageType)
	binary.BigEndian.PutUint32(routeLenBuf, header.RouteLen)

	// Write header into buffer(except for dataLen)
	headerBuffer.Write([]byte(header.Sender))
	headerBuffer.Write([]byte(header.Accepter))
	headerBuffer.Write(messageTypeBuf)
	headerBuffer.Write(routeLenBuf)
	headerBuffer.Write([]byte(header.Route))

	// Check if message's data is needed to encrypt
	if !isPass {
		switch header.MessageType {
		case HI:
			mmess := mess.(*HIMess)
			greetingLenBuf := make([]byte, 2)
			binary.BigEndian.PutUint16(greetingLenBuf, mmess.GreetingLen)

			greetingBuf := []byte(mmess.Greeting)

			uuidLenBuf := make([]byte, 2)
			binary.BigEndian.PutUint16(uuidLenBuf, mmess.UUIDLen)

			uuidBuf := []byte(mmess.UUID)

			isAdminBuf := make([]byte, 2)
			binary.BigEndian.PutUint16(isAdminBuf, mmess.IsAdmin)

			isReconnectBuf := make([]byte, 2)
			binary.BigEndian.PutUint16(isReconnectBuf, mmess.IsReconnect)

			// Collect all spilted data, try encrypt them
			// use message.DataBuffer directly to save memory
			dataBuffer.Write(greetingLenBuf)
			dataBuffer.Write(greetingBuf)
			dataBuffer.Write(uuidLenBuf)
			dataBuffer.Write(uuidBuf)
			dataBuffer.Write(isAdminBuf)
			dataBuffer.Write(isReconnectBuf)
		case UUID:
			mmess := mess.(*UUIDMess)
			uuidLenBuf := make([]byte, 2)
			binary.BigEndian.PutUint16(uuidLenBuf, mmess.UUIDLen)

			uuidBuf := []byte(mmess.UUID)

			dataBuffer.Write(uuidLenBuf)
			dataBuffer.Write(uuidBuf)
		case CHILDUUIDREQ:
			mmess := mess.(*ChildUUIDReq)
			puuidLenBuf := make([]byte, 2)
			binary.BigEndian.PutUint16(puuidLenBuf, mmess.ParentUUIDLen)

			puuidBuf := []byte(mmess.ParentUUID)

			ipLenBuf := make([]byte, 2)
			binary.BigEndian.PutUint16(ipLenBuf, mmess.IPLen)

			ipBuf := []byte(mmess.IP)

			dataBuffer.Write(puuidLenBuf)
			dataBuffer.Write(puuidBuf)
			dataBuffer.Write(ipLenBuf)
			dataBuffer.Write(ipBuf)
		case CHILDUUIDRES:
			mmess := mess.(*ChildUUIDRes)
			uuidLenBuf := make([]byte, 2)
			binary.BigEndian.PutUint16(uuidLenBuf, mmess.UUIDLen)

			uuidBuf := []byte(mmess.UUID)

			dataBuffer.Write(uuidLenBuf)
			dataBuffer.Write(uuidBuf)
		case MYINFO:
			mmess := mess.(*MyInfo)
			uuidLenBuf := make([]byte, 2)
			binary.BigEndian.PutUint16(uuidLenBuf, mmess.UUIDLen)

			uuidBuf := []byte(mmess.UUID)

			usernameLenBuf := make([]byte, 8)
			binary.BigEndian.PutUint64(usernameLenBuf, mmess.UsernameLen)

			usernameBuf := []byte(mmess.Username)

			hostnameLenBuf := make([]byte, 8)
			binary.BigEndian.PutUint64(hostnameLenBuf, mmess.HostnameLen)

			hostnameBuf := []byte(mmess.Hostname)

			memoLenBuf := make([]byte, 8)
			binary.BigEndian.PutUint64(memoLenBuf, mmess.MemoLen)

			memoBuf := []byte(mmess.Memo)

			dataBuffer.Write(uuidLenBuf)
			dataBuffer.Write(uuidBuf)
			dataBuffer.Write(usernameLenBuf)
			dataBuffer.Write(usernameBuf)
			dataBuffer.Write(hostnameLenBuf)
			dataBuffer.Write(hostnameBuf)
			dataBuffer.Write(memoLenBuf)
			dataBuffer.Write(memoBuf)
		case MYMEMO:
			mmess := mess.(*MyMemo)
			memoLenBuf := make([]byte, 8)
			binary.BigEndian.PutUint64(memoLenBuf, mmess.MemoLen)

			memoBuf := []byte(mmess.Memo)

			dataBuffer.Write(memoLenBuf)
			dataBuffer.Write(memoBuf)
		case SHELLREQ:
			mmess := mess.(*ShellReq)
			startBuf := make([]byte, 2)
			binary.BigEndian.PutUint16(startBuf, mmess.Start)

			dataBuffer.Write(startBuf)
		case SHELLRES:
			mmess := mess.(*ShellRes)
			OKBuf := make([]byte, 2)
			binary.BigEndian.PutUint16(OKBuf, mmess.OK)

			dataBuffer.Write(OKBuf)
		case SHELLCOMMAND:
			mmess := mess.(*ShellCommand)
			commandLenBuf := make([]byte, 8)
			binary.BigEndian.PutUint64(commandLenBuf, mmess.CommandLen)

			commandBuf := []byte(mmess.Command)

			dataBuffer.Write(commandLenBuf)
			dataBuffer.Write(commandBuf)
		case SHELLRESULT:
			mmess := mess.(*ShellResult)

			resultLenBuf := make([]byte, 8)
			binary.BigEndian.PutUint64(resultLenBuf, mmess.ResultLen)

			resultBuf := []byte(mmess.Result)

			dataBuffer.Write(resultLenBuf)
			dataBuffer.Write(resultBuf)
		case SHELLEXIT:
			mmess := mess.(*ShellExit)
			OKBuf := make([]byte, 2)
			binary.BigEndian.PutUint16(OKBuf, mmess.OK)

			dataBuffer.Write(OKBuf)
		case LISTENREQ:
			mmess := mess.(*ListenReq)
			methodBuf := make([]byte, 2)
			binary.BigEndian.PutUint16(methodBuf, mmess.Method)

			addrLenBuf := make([]byte, 8)
			binary.BigEndian.PutUint64(addrLenBuf, mmess.AddrLen)

			addrBuf := []byte(mmess.Addr)

			dataBuffer.Write(methodBuf)
			dataBuffer.Write(addrLenBuf)
			dataBuffer.Write(addrBuf)
		case LISTENRES:
			mmess := mess.(*ListenRes)
			OKBuf := make([]byte, 2)
			binary.BigEndian.PutUint16(OKBuf, mmess.OK)

			dataBuffer.Write(OKBuf)
		case SSHREQ:
			mmess := mess.(*SSHReq)
			methodBuf := make([]byte, 2)
			binary.BigEndian.PutUint16(methodBuf, mmess.Method)

			addrLenBuf := make([]byte, 2)
			binary.BigEndian.PutUint16(addrLenBuf, mmess.AddrLen)

			addrBuf := []byte(mmess.Addr)

			usernameLenBuf := make([]byte, 8)
			binary.BigEndian.PutUint64(usernameLenBuf, mmess.UsernameLen)

			usernameBuf := []byte(mmess.Username)

			passwordLenBuf := make([]byte, 8)
			binary.BigEndian.PutUint64(passwordLenBuf, mmess.PasswordLen)

			passwordBuf := []byte(mmess.Password)

			certificateLenBuf := make([]byte, 8)
			binary.BigEndian.PutUint64(certificateLenBuf, mmess.CertificateLen)

			certificateBuf := mmess.Certificate

			dataBuffer.Write(methodBuf)
			dataBuffer.Write(addrLenBuf)
			dataBuffer.Write(addrBuf)
			dataBuffer.Write(usernameLenBuf)
			dataBuffer.Write(usernameBuf)
			dataBuffer.Write(passwordLenBuf)
			dataBuffer.Write(passwordBuf)
			dataBuffer.Write(certificateLenBuf)
			dataBuffer.Write(certificateBuf)
		case SSHRES:
			mmess := mess.(*SSHRes)
			OKBuf := make([]byte, 2)
			binary.BigEndian.PutUint16(OKBuf, mmess.OK)

			dataBuffer.Write(OKBuf)
		case SSHCOMMAND:
			mmess := mess.(*SSHCommand)

			commandLenBuf := make([]byte, 8)
			binary.BigEndian.PutUint64(commandLenBuf, mmess.CommandLen)

			commandBuf := []byte(mmess.Command)

			dataBuffer.Write(commandLenBuf)
			dataBuffer.Write(commandBuf)
		case SSHRESULT:
			mmess := mess.(*SSHResult)

			resultLenBuf := make([]byte, 8)
			binary.BigEndian.PutUint64(resultLenBuf, mmess.ResultLen)

			resultBuf := []byte(mmess.Result)

			dataBuffer.Write(resultLenBuf)
			dataBuffer.Write(resultBuf)
		case SSHEXIT:
			mmess := mess.(*SSHExit)
			OKBuf := make([]byte, 2)
			binary.BigEndian.PutUint16(OKBuf, mmess.OK)

			dataBuffer.Write(OKBuf)
		case SSHTUNNELREQ:
			mmess := mess.(*SSHTunnelReq)
			methodBuf := make([]byte, 2)
			binary.BigEndian.PutUint16(methodBuf, mmess.Method)

			addrLenBuf := make([]byte, 2)
			binary.BigEndian.PutUint16(addrLenBuf, mmess.AddrLen)

			addrBuf := []byte(mmess.Addr)

			portLenBuf := make([]byte, 2)
			binary.BigEndian.PutUint16(portLenBuf, mmess.PortLen)

			portBuf := []byte(mmess.Port)

			usernameLenBuf := make([]byte, 8)
			binary.BigEndian.PutUint64(usernameLenBuf, mmess.UsernameLen)

			usernameBuf := []byte(mmess.Username)

			passwordLenBuf := make([]byte, 8)
			binary.BigEndian.PutUint64(passwordLenBuf, mmess.PasswordLen)

			passwordBuf := []byte(mmess.Password)

			certificateLenBuf := make([]byte, 8)
			binary.BigEndian.PutUint64(certificateLenBuf, mmess.CertificateLen)

			certificateBuf := mmess.Certificate

			dataBuffer.Write(methodBuf)
			dataBuffer.Write(addrLenBuf)
			dataBuffer.Write(addrBuf)
			dataBuffer.Write(portLenBuf)
			dataBuffer.Write(portBuf)
			dataBuffer.Write(usernameLenBuf)
			dataBuffer.Write(usernameBuf)
			dataBuffer.Write(passwordLenBuf)
			dataBuffer.Write(passwordBuf)
			dataBuffer.Write(certificateLenBuf)
			dataBuffer.Write(certificateBuf)
		case SSHTUNNELRES:
			mmess := mess.(*SSHTunnelRes)
			OKBuf := make([]byte, 2)
			binary.BigEndian.PutUint16(OKBuf, mmess.OK)

			dataBuffer.Write(OKBuf)
		case FILESTATREQ:
			mmess := mess.(*FileStatReq)

			filenameLenBuf := make([]byte, 4)
			binary.BigEndian.PutUint32(filenameLenBuf, mmess.FilenameLen)

			filenameBuf := []byte(mmess.Filename)

			fileSizeBuf := make([]byte, 8)
			binary.BigEndian.PutUint64(fileSizeBuf, mmess.FileSize)

			sliceNumBuf := make([]byte, 8)
			binary.BigEndian.PutUint64(sliceNumBuf, mmess.SliceNum)

			dataBuffer.Write(filenameLenBuf)
			dataBuffer.Write(filenameBuf)
			dataBuffer.Write(fileSizeBuf)
			dataBuffer.Write(sliceNumBuf)
		case FILESTATRES:
			mmess := mess.(*FileStatRes)
			OKBuf := make([]byte, 2)
			binary.BigEndian.PutUint16(OKBuf, mmess.OK)

			dataBuffer.Write(OKBuf)
		case FILEDATA:
			mmess := mess.(*FileData)
			dataLenBuf := make([]byte, 8)
			binary.BigEndian.PutUint64(dataLenBuf, mmess.DataLen)

			dataBuf := mmess.Data

			dataBuffer.Write(dataLenBuf)
			dataBuffer.Write(dataBuf)
		case FILEERR:
			mmess := mess.(*FileErr)
			errorBuf := make([]byte, 2)
			binary.BigEndian.PutUint16(errorBuf, mmess.Error)

			dataBuffer.Write(errorBuf)
		case FILEDOWNREQ:
			mmess := mess.(*FileDownReq)

			filePathLenBuf := make([]byte, 4)
			binary.BigEndian.PutUint32(filePathLenBuf, mmess.FilePathLen)

			filePathBuf := []byte(mmess.FilePath)

			filenameLenBuf := make([]byte, 4)
			binary.BigEndian.PutUint32(filenameLenBuf, mmess.FilenameLen)

			filenameBuf := []byte(mmess.Filename)

			dataBuffer.Write(filePathLenBuf)
			dataBuffer.Write(filePathBuf)
			dataBuffer.Write(filenameLenBuf)
			dataBuffer.Write(filenameBuf)
		case FILEDOWNRES:
			mmess := mess.(*FileDownRes)
			OKBuf := make([]byte, 2)
			binary.BigEndian.PutUint16(OKBuf, mmess.OK)

			dataBuffer.Write(OKBuf)
		case SOCKSSTART:
			mmess := mess.(*SocksStart)
			usernameLenBuf := make([]byte, 8)
			binary.BigEndian.PutUint64(usernameLenBuf, mmess.UsernameLen)

			usernameBuf := []byte(mmess.Username)

			passwordLenBuf := make([]byte, 8)
			binary.BigEndian.PutUint64(passwordLenBuf, mmess.PasswordLen)

			passwordBuf := []byte(mmess.Password)

			dataBuffer.Write(usernameLenBuf)
			dataBuffer.Write(usernameBuf)
			dataBuffer.Write(passwordLenBuf)
			dataBuffer.Write(passwordBuf)
		case SOCKSTCPDATA:
			mmess := mess.(*SocksTCPData)
			seqBuf := make([]byte, 8)
			binary.BigEndian.PutUint64(seqBuf, mmess.Seq)

			dataLenBuf := make([]byte, 8)
			binary.BigEndian.PutUint64(dataLenBuf, mmess.DataLen)

			dataBuf := mmess.Data

			dataBuffer.Write(seqBuf)
			dataBuffer.Write(dataLenBuf)
			dataBuffer.Write(dataBuf)
		case SOCKSUDPDATA:
			mmess := mess.(*SocksUDPData)
			seqBuf := make([]byte, 8)
			binary.BigEndian.PutUint64(seqBuf, mmess.Seq)

			dataLenBuf := make([]byte, 8)
			binary.BigEndian.PutUint64(dataLenBuf, mmess.DataLen)

			dataBuf := mmess.Data

			dataBuffer.Write(seqBuf)
			dataBuffer.Write(dataLenBuf)
			dataBuffer.Write(dataBuf)
		case UDPASSSTART:
			mmess := mess.(*UDPAssStart)
			seqBuf := make([]byte, 8)
			binary.BigEndian.PutUint64(seqBuf, mmess.Seq)

			sourceAddrLenBuf := make([]byte, 2)
			binary.BigEndian.PutUint16(sourceAddrLenBuf, mmess.SourceAddrLen)

			sourceAddrBuf := []byte(mmess.SourceAddr)

			dataBuffer.Write(seqBuf)
			dataBuffer.Write(sourceAddrLenBuf)
			dataBuffer.Write(sourceAddrBuf)
		case UDPASSRES:
			mmess := mess.(*UDPAssRes)
			seqBuf := make([]byte, 8)
			binary.BigEndian.PutUint64(seqBuf, mmess.Seq)

			OKBuf := make([]byte, 2)
			binary.BigEndian.PutUint16(OKBuf, mmess.OK)

			addrLenBuf := make([]byte, 2)
			binary.BigEndian.PutUint16(addrLenBuf, mmess.AddrLen)

			addrBuf := []byte(mmess.Addr)

			dataBuffer.Write(seqBuf)
			dataBuffer.Write(OKBuf)
			dataBuffer.Write(addrLenBuf)
			dataBuffer.Write(addrBuf)
		case SOCKSTCPFIN:
			mmess := mess.(*SocksTCPFin)
			seqBuf := make([]byte, 8)
			binary.BigEndian.PutUint64(seqBuf, mmess.Seq)

			dataBuffer.Write(seqBuf)
		case SOCKSREADY:
			mmess := mess.(*SocksReady)
			OKBuf := make([]byte, 2)
			binary.BigEndian.PutUint16(OKBuf, mmess.OK)

			dataBuffer.Write(OKBuf)
		case FORWARDTEST:
			mmess := mess.(*ForwardTest)

			addrLenBuf := make([]byte, 2)
			binary.BigEndian.PutUint16(addrLenBuf, mmess.AddrLen)

			addrBuf := []byte(mmess.Addr)

			dataBuffer.Write(addrLenBuf)
			dataBuffer.Write(addrBuf)
		case FORWARDSTART:
			mmess := mess.(*ForwardStart)

			seqBuf := make([]byte, 8)
			binary.BigEndian.PutUint64(seqBuf, mmess.Seq)

			addrLenBuf := make([]byte, 2)
			binary.BigEndian.PutUint16(addrLenBuf, mmess.AddrLen)

			addrBuf := []byte(mmess.Addr)

			dataBuffer.Write(seqBuf)
			dataBuffer.Write(addrLenBuf)
			dataBuffer.Write(addrBuf)
		case FORWARDREADY:
			mmess := mess.(*ForwardReady)
			OKBuf := make([]byte, 2)
			binary.BigEndian.PutUint16(OKBuf, mmess.OK)

			dataBuffer.Write(OKBuf)
		case FORWARDDATA:
			mmess := mess.(*ForwardData)
			seqBuf := make([]byte, 8)
			binary.BigEndian.PutUint64(seqBuf, mmess.Seq)

			dataLenBuf := make([]byte, 8)
			binary.BigEndian.PutUint64(dataLenBuf, mmess.DataLen)

			dataBuf := mmess.Data

			dataBuffer.Write(seqBuf)
			dataBuffer.Write(dataLenBuf)
			dataBuffer.Write(dataBuf)
		case FORWARDFIN:
			mmess := mess.(*ForwardFin)
			seqBuf := make([]byte, 8)
			binary.BigEndian.PutUint64(seqBuf, mmess.Seq)

			dataBuffer.Write(seqBuf)
		case BACKWARDTEST:
			mmess := mess.(*BackwardTest)

			lPortLenBuf := make([]byte, 2)
			binary.BigEndian.PutUint16(lPortLenBuf, mmess.LPortLen)

			lPortBuf := []byte(mmess.LPort)

			rPortLenBuf := make([]byte, 2)
			binary.BigEndian.PutUint16(rPortLenBuf, mmess.RPortLen)

			rPortBuf := []byte(mmess.RPort)

			dataBuffer.Write(lPortLenBuf)
			dataBuffer.Write(lPortBuf)
			dataBuffer.Write(rPortLenBuf)
			dataBuffer.Write(rPortBuf)
		case BACKWARDREADY:
			mmess := mess.(*BackwardReady)
			OKBuf := make([]byte, 2)
			binary.BigEndian.PutUint16(OKBuf, mmess.OK)

			dataBuffer.Write(OKBuf)
		case BACKWARDSTART:
			mmess := mess.(*BackwardStart)
			uuidLenBuf := make([]byte, 2)
			binary.BigEndian.PutUint16(uuidLenBuf, mmess.UUIDLen)

			uuidBuf := []byte(mmess.UUID)

			lPortLenBuf := make([]byte, 2)
			binary.BigEndian.PutUint16(lPortLenBuf, mmess.LPortLen)

			lPortBuf := []byte(mmess.LPort)

			rPortLenBuf := make([]byte, 2)
			binary.BigEndian.PutUint16(rPortLenBuf, mmess.RPortLen)

			rPortBuf := []byte(mmess.RPort)

			dataBuffer.Write(uuidLenBuf)
			dataBuffer.Write(uuidBuf)
			dataBuffer.Write(lPortLenBuf)
			dataBuffer.Write(lPortBuf)
			dataBuffer.Write(rPortLenBuf)
			dataBuffer.Write(rPortBuf)
		case BACKWARDSEQ:
			mmess := mess.(*BackwardSeq)
			seqBuf := make([]byte, 8)
			binary.BigEndian.PutUint64(seqBuf, mmess.Seq)

			rPortLenBuf := make([]byte, 2)
			binary.BigEndian.PutUint16(rPortLenBuf, mmess.RPortLen)

			rPortBuf := []byte(mmess.RPort)

			dataBuffer.Write(seqBuf)
			dataBuffer.Write(rPortLenBuf)
			dataBuffer.Write(rPortBuf)
		case BACKWARDDATA:
			mmess := mess.(*BackwardData)
			seqBuf := make([]byte, 8)
			binary.BigEndian.PutUint64(seqBuf, mmess.Seq)

			dataLenBuf := make([]byte, 8)
			binary.BigEndian.PutUint64(dataLenBuf, mmess.DataLen)

			dataBuf := mmess.Data

			dataBuffer.Write(seqBuf)
			dataBuffer.Write(dataLenBuf)
			dataBuffer.Write(dataBuf)
		case BACKWARDFIN:
			mmess := mess.(*BackWardFin)
			seqBuf := make([]byte, 8)
			binary.BigEndian.PutUint64(seqBuf, mmess.Seq)

			dataBuffer.Write(seqBuf)
		case BACKWARDSTOP:
			mmess := mess.(*BackwardStop)
			allBuf := make([]byte, 2)
			binary.BigEndian.PutUint16(allBuf, mmess.All)

			rPortLenBuf := make([]byte, 2)
			binary.BigEndian.PutUint16(rPortLenBuf, mmess.RPortLen)

			rPortBuf := []byte(mmess.RPort)

			dataBuffer.Write(allBuf)
			dataBuffer.Write(rPortLenBuf)
			dataBuffer.Write(rPortBuf)
		case BACKWARDSTOPDONE:
			mmess := mess.(*BackwardStopDone)
			allBuf := make([]byte, 2)
			binary.BigEndian.PutUint16(allBuf, mmess.All)

			uuidLenBuf := make([]byte, 2)
			binary.BigEndian.PutUint16(uuidLenBuf, mmess.UUIDLen)

			uuidBuf := []byte(mmess.UUID)

			rPortLenBuf := make([]byte, 2)
			binary.BigEndian.PutUint16(rPortLenBuf, mmess.RPortLen)

			rPortBuf := []byte(mmess.RPort)

			dataBuffer.Write(allBuf)
			dataBuffer.Write(uuidLenBuf)
			dataBuffer.Write(uuidBuf)
			dataBuffer.Write(rPortLenBuf)
			dataBuffer.Write(rPortBuf)
		case CONNECTSTART:
			mmess := mess.(*ConnectStart)

			addrLenBuf := make([]byte, 2)
			binary.BigEndian.PutUint16(addrLenBuf, mmess.AddrLen)

			addrBuf := []byte(mmess.Addr)

			dataBuffer.Write(addrLenBuf)
			dataBuffer.Write(addrBuf)
		case CONNECTDONE:
			mmess := mess.(*ConnectDone)
			OKBuf := make([]byte, 2)
			binary.BigEndian.PutUint16(OKBuf, mmess.OK)

			dataBuffer.Write(OKBuf)
		case NODEOFFLINE:
			mmess := mess.(*NodeOffline)

			uuidLenBuf := make([]byte, 2)
			binary.BigEndian.PutUint16(uuidLenBuf, mmess.UUIDLen)

			uuidBuf := []byte(mmess.UUID)

			dataBuffer.Write(uuidLenBuf)
			dataBuffer.Write(uuidBuf)
		case NODEREONLINE:
			mmess := mess.(*NodeReonline)

			puuidLenBuf := make([]byte, 2)
			binary.BigEndian.PutUint16(puuidLenBuf, mmess.ParentUUIDLen)

			puuidBuf := []byte(mmess.ParentUUID)

			uuidLenBuf := make([]byte, 2)
			binary.BigEndian.PutUint16(uuidLenBuf, mmess.UUIDLen)

			uuidBuf := []byte(mmess.UUID)

			ipLenBuf := make([]byte, 2)
			binary.BigEndian.PutUint16(ipLenBuf, mmess.IPLen)

			ipBuf := []byte(mmess.IP)

			dataBuffer.Write(puuidLenBuf)
			dataBuffer.Write(puuidBuf)
			dataBuffer.Write(uuidLenBuf)
			dataBuffer.Write(uuidBuf)
			dataBuffer.Write(ipLenBuf)
			dataBuffer.Write(ipBuf)
		case UPSTREAMOFFLINE:
			mmess := mess.(*UpstreamOffline)
			OKBuf := make([]byte, 2)
			binary.BigEndian.PutUint16(OKBuf, mmess.OK)

			dataBuffer.Write(OKBuf)
		case UPSTREAMREONLINE:
			mmess := mess.(*UpstreamReonline)
			OKBuf := make([]byte, 2)
			binary.BigEndian.PutUint16(OKBuf, mmess.OK)

			dataBuffer.Write(OKBuf)
		case SHUTDOWN:
			mmess := mess.(*Shutdown)
			OKBuf := make([]byte, 2)
			binary.BigEndian.PutUint16(OKBuf, mmess.OK)

			dataBuffer.Write(OKBuf)
		default:
		}
	} else {
		mmess := mess.([]byte)
		dataBuffer.Write(mmess)
	}

	// Encrypt data
	message.DataBuffer = dataBuffer.Bytes()

	if !isPass {
		message.DataBuffer = crypto.AESEncrypt(message.DataBuffer, message.CryptoSecret)
	}
	// Calculate the whole data's length
	dataLenBuf := make([]byte, 8)
	binary.BigEndian.PutUint64(dataLenBuf, uint64(len(message.DataBuffer)))
	headerBuffer.Write(dataLenBuf)
	message.HeaderBuffer = headerBuffer.Bytes()
}

func (message *RawMessage) ConstructSuffix() {}

func (message *RawMessage) DeconstructHeader() {}

func (message *RawMessage) DeconstructData() (*Header, interface{}, error) {
	var (
		header         = new(Header)
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

	if header.Accepter == TEMP_UUID || message.UUID == ADMIN_UUID || message.UUID == header.Accepter {
		dataBuf = crypto.AESDecrypt(dataBuf[:], message.CryptoSecret) // use dataBuf directly to save the memory
	} else {
		return header, dataBuf, nil
	}

	switch header.MessageType {
	case HI:
		mmess := new(HIMess)
		mmess.GreetingLen = binary.BigEndian.Uint16(dataBuf[:2])
		mmess.Greeting = string(dataBuf[2 : 2+mmess.GreetingLen])
		mmess.UUIDLen = binary.BigEndian.Uint16(dataBuf[2+mmess.GreetingLen : 4+mmess.GreetingLen])
		mmess.UUID = string(dataBuf[4+mmess.GreetingLen : 4+mmess.GreetingLen+mmess.UUIDLen])
		mmess.IsAdmin = binary.BigEndian.Uint16(dataBuf[4+mmess.GreetingLen+mmess.UUIDLen : 6+mmess.GreetingLen+mmess.UUIDLen])
		mmess.IsReconnect = binary.BigEndian.Uint16(dataBuf[6+mmess.GreetingLen+mmess.UUIDLen : 8+mmess.GreetingLen+mmess.UUIDLen])
		return header, mmess, nil
	case UUID:
		mmess := new(UUIDMess)
		mmess.UUIDLen = binary.BigEndian.Uint16(dataBuf[:2])
		mmess.UUID = string(dataBuf[2 : 2+mmess.UUIDLen])
		return header, mmess, nil
	case CHILDUUIDREQ:
		mmess := new(ChildUUIDReq)
		mmess.ParentUUIDLen = binary.BigEndian.Uint16(dataBuf[:2])
		mmess.ParentUUID = string(dataBuf[2 : 2+mmess.ParentUUIDLen])
		mmess.IPLen = binary.BigEndian.Uint16(dataBuf[2+mmess.ParentUUIDLen : 4+mmess.ParentUUIDLen])
		mmess.IP = string(dataBuf[4+mmess.ParentUUIDLen : 4+mmess.ParentUUIDLen+mmess.IPLen])
		return header, mmess, nil
	case CHILDUUIDRES:
		mmess := new(ChildUUIDRes)
		mmess.UUIDLen = binary.BigEndian.Uint16(dataBuf[:2])
		mmess.UUID = string(dataBuf[2 : 2+mmess.UUIDLen])
		return header, mmess, nil
	case MYINFO:
		mmess := new(MyInfo)
		mmess.UUIDLen = binary.BigEndian.Uint16(dataBuf[:2])
		mmess.UUID = string(dataBuf[2 : 2+mmess.UUIDLen])
		mmess.UsernameLen = binary.BigEndian.Uint64(dataBuf[2+mmess.UUIDLen : 10+mmess.UUIDLen])
		mmess.Username = string(dataBuf[10+mmess.UUIDLen : 10+uint64(mmess.UUIDLen)+mmess.UsernameLen])
		mmess.HostnameLen = binary.BigEndian.Uint64(dataBuf[10+uint64(mmess.UUIDLen)+mmess.UsernameLen : 18+uint64(mmess.UUIDLen)+mmess.UsernameLen])
		mmess.Hostname = string(dataBuf[18+uint64(mmess.UUIDLen)+mmess.UsernameLen : 18+uint64(mmess.UUIDLen)+mmess.UsernameLen+mmess.HostnameLen])
		mmess.MemoLen = binary.BigEndian.Uint64(dataBuf[18+uint64(mmess.UUIDLen)+mmess.UsernameLen+mmess.HostnameLen : 26+uint64(mmess.UUIDLen)+mmess.UsernameLen+mmess.HostnameLen])
		mmess.Memo = string(dataBuf[26+uint64(mmess.UUIDLen)+mmess.UsernameLen+mmess.HostnameLen : 26+uint64(mmess.UUIDLen)+mmess.UsernameLen+mmess.HostnameLen+mmess.MemoLen])
		return header, mmess, nil
	case MYMEMO:
		mmess := new(MyMemo)
		mmess.MemoLen = binary.BigEndian.Uint64(dataBuf[:8])
		mmess.Memo = string(dataBuf[8 : 8+mmess.MemoLen])
		return header, mmess, nil
	case SHELLREQ:
		mmess := new(ShellReq)
		mmess.Start = binary.BigEndian.Uint16(dataBuf[:2])
		return header, mmess, nil
	case SHELLRES:
		mmess := new(ShellRes)
		mmess.OK = binary.BigEndian.Uint16(dataBuf[:2])
		return header, mmess, nil
	case SHELLCOMMAND:
		mmess := new(ShellCommand)
		mmess.CommandLen = binary.BigEndian.Uint64(dataBuf[:8])
		mmess.Command = string(dataBuf[8 : 8+mmess.CommandLen])
		return header, mmess, nil
	case SHELLRESULT:
		mmess := new(ShellResult)
		mmess.ResultLen = binary.BigEndian.Uint64(dataBuf[:8])
		mmess.Result = string(dataBuf[8 : 8+mmess.ResultLen])
		return header, mmess, nil
	case SHELLEXIT:
		mmess := new(ShellExit)
		mmess.OK = binary.BigEndian.Uint16(dataBuf[:2])
		return header, mmess, nil
	case LISTENREQ:
		mmess := new(ListenReq)
		mmess.Method = binary.BigEndian.Uint16(dataBuf[:2])
		mmess.AddrLen = binary.BigEndian.Uint64(dataBuf[2:10])
		mmess.Addr = string(dataBuf[10 : 10+mmess.AddrLen])
		return header, mmess, nil
	case LISTENRES:
		mmess := new(ListenRes)
		mmess.OK = binary.BigEndian.Uint16(dataBuf[:2])
		return header, mmess, nil
	case SSHREQ:
		mmess := new(SSHReq)
		mmess.Method = binary.BigEndian.Uint16(dataBuf[:2])
		mmess.AddrLen = binary.BigEndian.Uint16(dataBuf[2:4])
		mmess.Addr = string(dataBuf[4 : 4+mmess.AddrLen])
		mmess.UsernameLen = binary.BigEndian.Uint64(dataBuf[4+mmess.AddrLen : 12+mmess.AddrLen])
		mmess.Username = string(dataBuf[12+mmess.AddrLen : 12+uint64(mmess.AddrLen)+mmess.UsernameLen])
		mmess.PasswordLen = binary.BigEndian.Uint64(dataBuf[12+uint64(mmess.AddrLen)+mmess.UsernameLen : 20+uint64(mmess.AddrLen)+mmess.UsernameLen])
		mmess.Password = string(dataBuf[20+uint64(mmess.AddrLen)+mmess.UsernameLen : 20+uint64(mmess.AddrLen)+mmess.UsernameLen+mmess.PasswordLen])
		mmess.CertificateLen = binary.BigEndian.Uint64(dataBuf[20+uint64(mmess.AddrLen)+mmess.UsernameLen+mmess.PasswordLen : 28+uint64(mmess.AddrLen)+mmess.UsernameLen+mmess.PasswordLen])
		mmess.Certificate = dataBuf[28+uint64(mmess.AddrLen)+mmess.UsernameLen+mmess.PasswordLen : 28+uint64(mmess.AddrLen)+mmess.UsernameLen+mmess.PasswordLen+mmess.CertificateLen]
		return header, mmess, nil
	case SSHRES:
		mmess := new(SSHRes)
		mmess.OK = binary.BigEndian.Uint16(dataBuf[:2])
		return header, mmess, nil
	case SSHCOMMAND:
		mmess := new(SSHCommand)
		mmess.CommandLen = binary.BigEndian.Uint64(dataBuf[:8])
		mmess.Command = string(dataBuf[8 : 8+mmess.CommandLen])
		return header, mmess, nil
	case SSHRESULT:
		mmess := new(SSHResult)
		mmess.ResultLen = binary.BigEndian.Uint64(dataBuf[:8])
		mmess.Result = string(dataBuf[8 : 8+mmess.ResultLen])
		return header, mmess, nil
	case SSHEXIT:
		mmess := new(SSHExit)
		mmess.OK = binary.BigEndian.Uint16(dataBuf[:2])
		return header, mmess, nil
	case SSHTUNNELREQ:
		mmess := new(SSHTunnelReq)
		mmess.Method = binary.BigEndian.Uint16(dataBuf[:2])
		mmess.AddrLen = binary.BigEndian.Uint16(dataBuf[2:4])
		mmess.Addr = string(dataBuf[4 : 4+mmess.AddrLen])
		mmess.PortLen = binary.BigEndian.Uint16(dataBuf[4+mmess.AddrLen : 6+mmess.AddrLen])
		mmess.Port = string(dataBuf[6+mmess.AddrLen : 6+mmess.AddrLen+mmess.PortLen])
		mmess.UsernameLen = binary.BigEndian.Uint64(dataBuf[6+mmess.AddrLen+mmess.PortLen : 14+mmess.AddrLen+mmess.PortLen])
		mmess.Username = string(dataBuf[14+mmess.AddrLen+mmess.PortLen : 14+uint64(mmess.AddrLen)+uint64(mmess.PortLen)+mmess.UsernameLen])
		mmess.PasswordLen = binary.BigEndian.Uint64(dataBuf[14+uint64(mmess.AddrLen)+uint64(mmess.PortLen)+mmess.UsernameLen : 22+uint64(mmess.AddrLen)+uint64(mmess.PortLen)+mmess.UsernameLen])
		mmess.Password = string(dataBuf[22+uint64(mmess.AddrLen)+uint64(mmess.PortLen)+mmess.UsernameLen : 22+uint64(mmess.AddrLen)+uint64(mmess.PortLen)+mmess.UsernameLen+mmess.PasswordLen])
		mmess.CertificateLen = binary.BigEndian.Uint64(dataBuf[22+uint64(mmess.AddrLen)+uint64(mmess.PortLen)+mmess.UsernameLen+mmess.PasswordLen : 30+uint64(mmess.AddrLen)+uint64(mmess.PortLen)+mmess.UsernameLen+mmess.PasswordLen])
		mmess.Certificate = dataBuf[30+uint64(mmess.AddrLen)+uint64(mmess.PortLen)+mmess.UsernameLen+mmess.PasswordLen : 30+uint64(mmess.AddrLen)+uint64(mmess.PortLen)+mmess.UsernameLen+mmess.PasswordLen+mmess.CertificateLen]
		return header, mmess, nil
	case SSHTUNNELRES:
		mmess := new(SSHTunnelRes)
		mmess.OK = binary.BigEndian.Uint16(dataBuf[:2])
		return header, mmess, nil
	case FILESTATREQ:
		mmess := new(FileStatReq)
		mmess.FilenameLen = binary.BigEndian.Uint32(dataBuf[:4])
		mmess.Filename = string(dataBuf[4 : 4+mmess.FilenameLen])
		mmess.FileSize = binary.BigEndian.Uint64(dataBuf[4+mmess.FilenameLen : 12+mmess.FilenameLen])
		mmess.SliceNum = binary.BigEndian.Uint64(dataBuf[12+mmess.FilenameLen : 20+mmess.FilenameLen])
		return header, mmess, nil
	case FILESTATRES:
		mmess := new(FileStatRes)
		mmess.OK = binary.BigEndian.Uint16(dataBuf[:2])
		return header, mmess, nil
	case FILEDATA:
		mmess := new(FileData)
		mmess.DataLen = binary.BigEndian.Uint64(dataBuf[:8])
		mmess.Data = dataBuf[8 : 8+mmess.DataLen]
		return header, mmess, nil
	case FILEERR:
		mmess := new(FileErr)
		mmess.Error = binary.BigEndian.Uint16(dataBuf[:2])
		return header, mmess, nil
	case FILEDOWNREQ:
		mmess := new(FileDownReq)
		mmess.FilePathLen = binary.BigEndian.Uint32(dataBuf[:4])
		mmess.FilePath = string(dataBuf[4 : 4+mmess.FilePathLen])
		mmess.FilenameLen = binary.BigEndian.Uint32(dataBuf[4+mmess.FilePathLen : 8+mmess.FilePathLen])
		mmess.Filename = string(dataBuf[8+mmess.FilePathLen : 8+mmess.FilePathLen+mmess.FilenameLen])
		return header, mmess, nil
	case FILEDOWNRES:
		mmess := new(FileDownRes)
		mmess.OK = binary.BigEndian.Uint16(dataBuf[:2])
		return header, mmess, nil
	case SOCKSSTART:
		mmess := new(SocksStart)
		mmess.UsernameLen = binary.BigEndian.Uint64(dataBuf[:8])
		mmess.Username = string(dataBuf[8 : 8+mmess.UsernameLen])
		mmess.PasswordLen = binary.BigEndian.Uint64(dataBuf[8+mmess.UsernameLen : 16+mmess.UsernameLen])
		mmess.Password = string(dataBuf[16+mmess.UsernameLen : 16+mmess.UsernameLen+mmess.PasswordLen])
		return header, mmess, nil
	case SOCKSTCPDATA:
		mmess := new(SocksTCPData)
		mmess.Seq = binary.BigEndian.Uint64(dataBuf[:8])
		mmess.DataLen = binary.BigEndian.Uint64(dataBuf[8:16])
		mmess.Data = dataBuf[16 : 16+mmess.DataLen]
		return header, mmess, nil
	case SOCKSUDPDATA:
		mmess := new(SocksUDPData)
		mmess.Seq = binary.BigEndian.Uint64(dataBuf[:8])
		mmess.DataLen = binary.BigEndian.Uint64(dataBuf[8:16])
		mmess.Data = dataBuf[16 : 16+mmess.DataLen]
		return header, mmess, nil
	case UDPASSSTART:
		mmess := new(UDPAssStart)
		mmess.Seq = binary.BigEndian.Uint64(dataBuf[:8])
		mmess.SourceAddrLen = binary.BigEndian.Uint16(dataBuf[8:10])
		mmess.SourceAddr = string(dataBuf[10 : 10+mmess.SourceAddrLen])
		return header, mmess, nil
	case UDPASSRES:
		mmess := new(UDPAssRes)
		mmess.Seq = binary.BigEndian.Uint64(dataBuf[:8])
		mmess.OK = binary.BigEndian.Uint16(dataBuf[8:10])
		mmess.AddrLen = binary.BigEndian.Uint16(dataBuf[10:12])
		mmess.Addr = string(dataBuf[12 : 12+mmess.AddrLen])
		return header, mmess, nil
	case SOCKSTCPFIN:
		mmess := new(SocksTCPFin)
		mmess.Seq = binary.BigEndian.Uint64(dataBuf[:8])
		return header, mmess, nil
	case SOCKSREADY:
		mmess := new(SocksReady)
		mmess.OK = binary.BigEndian.Uint16(dataBuf[:2])
		return header, mmess, nil
	case FORWARDTEST:
		mmess := new(ForwardTest)
		mmess.AddrLen = binary.BigEndian.Uint16(dataBuf[:2])
		mmess.Addr = string(dataBuf[2 : 2+mmess.AddrLen])
		return header, mmess, nil
	case FORWARDSTART:
		mmess := new(ForwardStart)
		mmess.Seq = binary.BigEndian.Uint64(dataBuf[:8])
		mmess.AddrLen = binary.BigEndian.Uint16(dataBuf[8:10])
		mmess.Addr = string(dataBuf[10 : 10+mmess.AddrLen])
		return header, mmess, nil
	case FORWARDREADY:
		mmess := new(ForwardReady)
		mmess.OK = binary.BigEndian.Uint16(dataBuf[:2])
		return header, mmess, nil
	case FORWARDDATA:
		mmess := new(ForwardData)
		mmess.Seq = binary.BigEndian.Uint64(dataBuf[:8])
		mmess.DataLen = binary.BigEndian.Uint64(dataBuf[8:16])
		mmess.Data = dataBuf[16 : 16+mmess.DataLen]
		return header, mmess, nil
	case FORWARDFIN:
		mmess := new(ForwardFin)
		mmess.Seq = binary.BigEndian.Uint64(dataBuf[:8])
		return header, mmess, nil
	case BACKWARDTEST:
		mmess := new(BackwardTest)
		mmess.LPortLen = binary.BigEndian.Uint16(dataBuf[:2])
		mmess.LPort = string(dataBuf[2 : 2+mmess.LPortLen])
		mmess.RPortLen = binary.BigEndian.Uint16(dataBuf[2+mmess.LPortLen : 4+mmess.LPortLen])
		mmess.RPort = string(dataBuf[4+mmess.LPortLen : 4+mmess.LPortLen+mmess.RPortLen])
		return header, mmess, nil
	case BACKWARDREADY:
		mmess := new(BackwardReady)
		mmess.OK = binary.BigEndian.Uint16(dataBuf[:2])
		return header, mmess, nil
	case BACKWARDSTART:
		mmess := new(BackwardStart)
		mmess.UUIDLen = binary.BigEndian.Uint16(dataBuf[:2])
		mmess.UUID = string(dataBuf[2 : 2+mmess.UUIDLen])
		mmess.LPortLen = binary.BigEndian.Uint16(dataBuf[2+mmess.UUIDLen : 4+mmess.UUIDLen])
		mmess.LPort = string(dataBuf[4+mmess.UUIDLen : 4+mmess.UUIDLen+mmess.LPortLen])
		mmess.RPortLen = binary.BigEndian.Uint16(dataBuf[4+mmess.UUIDLen+mmess.LPortLen : 6+mmess.UUIDLen+mmess.LPortLen])
		mmess.RPort = string(dataBuf[6+mmess.UUIDLen+mmess.LPortLen : 6+mmess.UUIDLen+mmess.LPortLen+mmess.RPortLen])
		return header, mmess, nil
	case BACKWARDSEQ:
		mmess := new(BackwardSeq)
		mmess.Seq = binary.BigEndian.Uint64(dataBuf[0:8])
		mmess.RPortLen = binary.BigEndian.Uint16(dataBuf[8:10])
		mmess.RPort = string(dataBuf[10 : 10+mmess.RPortLen])
		return header, mmess, nil
	case BACKWARDDATA:
		mmess := new(BackwardData)
		mmess.Seq = binary.BigEndian.Uint64(dataBuf[:8])
		mmess.DataLen = binary.BigEndian.Uint64(dataBuf[8:16])
		mmess.Data = dataBuf[16 : 16+mmess.DataLen]
		return header, mmess, nil
	case BACKWARDFIN:
		mmess := new(BackWardFin)
		mmess.Seq = binary.BigEndian.Uint64(dataBuf[:8])
		return header, mmess, nil
	case BACKWARDSTOP:
		mmess := new(BackwardStop)
		mmess.All = binary.BigEndian.Uint16(dataBuf[0:2])
		mmess.RPortLen = binary.BigEndian.Uint16(dataBuf[2:4])
		mmess.RPort = string(dataBuf[4 : 4+mmess.RPortLen])
		return header, mmess, nil
	case BACKWARDSTOPDONE:
		mmess := new(BackwardStopDone)
		mmess.All = binary.BigEndian.Uint16(dataBuf[0:2])
		mmess.UUIDLen = binary.BigEndian.Uint16(dataBuf[2:4])
		mmess.UUID = string(dataBuf[4 : 4+mmess.UUIDLen])
		mmess.RPortLen = binary.BigEndian.Uint16(dataBuf[4+mmess.UUIDLen : 6+mmess.UUIDLen])
		mmess.RPort = string(dataBuf[6+mmess.UUIDLen : 6+mmess.UUIDLen+mmess.RPortLen])
		return header, mmess, nil
	case CONNECTSTART:
		mmess := new(ConnectStart)
		mmess.AddrLen = binary.BigEndian.Uint16(dataBuf[:2])
		mmess.Addr = string(dataBuf[2 : 2+mmess.AddrLen])
		return header, mmess, nil
	case CONNECTDONE:
		mmess := new(ConnectDone)
		mmess.OK = binary.BigEndian.Uint16(dataBuf[:2])
		return header, mmess, nil
	case NODEOFFLINE:
		mmess := new(NodeOffline)
		mmess.UUIDLen = binary.BigEndian.Uint16(dataBuf[0:2])
		mmess.UUID = string(dataBuf[2 : 2+mmess.UUIDLen])
		return header, mmess, nil
	case NODEREONLINE:
		mmess := new(NodeReonline)
		mmess.ParentUUIDLen = binary.BigEndian.Uint16(dataBuf[:2])
		mmess.ParentUUID = string(dataBuf[2 : 2+mmess.ParentUUIDLen])
		mmess.UUIDLen = binary.BigEndian.Uint16(dataBuf[2+mmess.ParentUUIDLen : 4+mmess.ParentUUIDLen])
		mmess.UUID = string(dataBuf[4+mmess.ParentUUIDLen : 4+mmess.ParentUUIDLen+mmess.UUIDLen])
		mmess.IPLen = binary.BigEndian.Uint16(dataBuf[4+mmess.ParentUUIDLen+mmess.UUIDLen : 6+mmess.ParentUUIDLen+mmess.UUIDLen])
		mmess.IP = string(dataBuf[6+mmess.ParentUUIDLen+mmess.UUIDLen : 6+mmess.ParentUUIDLen+mmess.UUIDLen+mmess.IPLen])
		return header, mmess, nil
	case UPSTREAMOFFLINE:
		mmess := new(UpstreamOffline)
		mmess.OK = binary.BigEndian.Uint16(dataBuf[:2])
		return header, mmess, nil
	case UPSTREAMREONLINE:
		mmess := new(UpstreamReonline)
		mmess.OK = binary.BigEndian.Uint16(dataBuf[:2])
		return header, mmess, nil
	case SHUTDOWN:
		mmess := new(Shutdown)
		mmess.OK = binary.BigEndian.Uint16(dataBuf[:2])
		return header, mmess, nil
	default:
	}

	return header, nil, errors.New("Unknown error!")
}

func (message *RawMessage) DeconstructSuffix() {}

func (message *RawMessage) SendMessage() {
	finalBuffer := append(message.HeaderBuffer, message.DataBuffer...)
	message.Conn.Write(finalBuffer)
	// Don't forget to set both Buffer to nil!!!
	message.HeaderBuffer = nil
	message.DataBuffer = nil
}
