package protocol

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"reflect"

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
		// Use reflect to construct data,optimize the code,thx to the idea from @lz520520
		messType := reflect.TypeOf(mess).Elem()
		messValue := reflect.ValueOf(mess).Elem()

		messFieldNum := messType.NumField()

		for i := 0; i < messFieldNum; i++ {
			inter := messValue.Field(i).Interface()

			switch value := inter.(type) {
			case string:
				dataBuffer.Write([]byte(value))
			case uint16:
				buffer := make([]byte, 2)
				binary.BigEndian.PutUint16(buffer, value)
				dataBuffer.Write(buffer)
			case uint32:
				buffer := make([]byte, 4)
				binary.BigEndian.PutUint32(buffer, value)
				dataBuffer.Write(buffer)
			case uint64:
				buffer := make([]byte, 8)
				binary.BigEndian.PutUint64(buffer, value)
				dataBuffer.Write(buffer)
			case []byte:
				dataBuffer.Write(value)
			}
		}
	} else {
		mmess := mess.([]byte)
		dataBuffer.Write(mmess)
	}

	message.DataBuffer = dataBuffer.Bytes()
	// Encrypt&Compress data
	if !isPass {
		message.DataBuffer = crypto.GzipCompress(message.DataBuffer)
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
	// Read header's element one by one
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
		dataBuf = crypto.AESDecrypt(dataBuf, message.CryptoSecret) // use dataBuf directly to save the memory
	} else {
		return header, dataBuf, nil
	}
	// Decompress the data
	dataBuf = crypto.GzipDecompress(dataBuf)
	// Use reflect to deconstruct data
	var mess interface{}
	switch header.MessageType {
	case HI:
		mess = new(HIMess)
	case UUID:
		mess = new(UUIDMess)
	case CHILDUUIDREQ:
		mess = new(ChildUUIDReq)
	case CHILDUUIDRES:
		mess = new(ChildUUIDRes)
	case MYINFO:
		mess = new(MyInfo)
	case MYMEMO:
		mess = new(MyMemo)
	case SHELLREQ:
		mess = new(ShellReq)
	case SHELLRES:
		mess = new(ShellRes)
	case SHELLCOMMAND:
		mess = new(ShellCommand)
	case SHELLRESULT:
		mess = new(ShellResult)
	case SHELLEXIT:
		mess = new(ShellExit)
	case LISTENREQ:
		mess = new(ListenReq)
	case LISTENRES:
		mess = new(ListenRes)
	case SSHREQ:
		mess = new(SSHReq)
	case SSHRES:
		mess = new(SSHRes)
	case SSHCOMMAND:
		mess = new(SSHCommand)
	case SSHRESULT:
		mess = new(SSHResult)
	case SSHEXIT:
		mess = new(SSHExit)
	case SSHTUNNELREQ:
		mess = new(SSHTunnelReq)
	case SSHTUNNELRES:
		mess = new(SSHTunnelRes)
	case FILESTATREQ:
		mess = new(FileStatReq)
	case FILESTATRES:
		mess = new(FileStatRes)
	case FILEDATA:
		mess = new(FileData)
	case FILEERR:
		mess = new(FileErr)
	case FILEDOWNREQ:
		mess = new(FileDownReq)
	case FILEDOWNRES:
		mess = new(FileDownRes)
	case SOCKSSTART:
		mess = new(SocksStart)
	case SOCKSTCPDATA:
		mess = new(SocksTCPData)
	case SOCKSUDPDATA:
		mess = new(SocksUDPData)
	case UDPASSSTART:
		mess = new(UDPAssStart)
	case UDPASSRES:
		mess = new(UDPAssRes)
	case SOCKSTCPFIN:
		mess = new(SocksTCPFin)
	case SOCKSREADY:
		mess = new(SocksReady)
	case FORWARDTEST:
		mess = new(ForwardTest)
	case FORWARDSTART:
		mess = new(ForwardStart)
	case FORWARDREADY:
		mess = new(ForwardReady)
	case FORWARDDATA:
		mess = new(ForwardData)
	case FORWARDFIN:
		mess = new(ForwardFin)
	case BACKWARDTEST:
		mess = new(BackwardTest)
	case BACKWARDREADY:
		mess = new(BackwardReady)
	case BACKWARDSTART:
		mess = new(BackwardStart)
	case BACKWARDSEQ:
		mess = new(BackwardSeq)
	case BACKWARDDATA:
		mess = new(BackwardData)
	case BACKWARDFIN:
		mess = new(BackWardFin)
	case BACKWARDSTOP:
		mess = new(BackwardStop)
	case BACKWARDSTOPDONE:
		mess = new(BackwardStopDone)
	case CONNECTSTART:
		mess = new(ConnectStart)
	case CONNECTDONE:
		mess = new(ConnectDone)
	case NODEOFFLINE:
		mess = new(NodeOffline)
	case NODEREONLINE:
		mess = new(NodeReonline)
	case UPSTREAMOFFLINE:
		mess = new(UpstreamOffline)
	case UPSTREAMREONLINE:
		mess = new(UpstreamReonline)
	case SHUTDOWN:
		mess = new(Shutdown)
	}

	messType := reflect.TypeOf(mess).Elem()
	messValue := reflect.ValueOf(mess).Elem()
	messFieldNum := messType.NumField()

	var ptr uint64
	for i := 0; i < messFieldNum; i++ {
		inter := messValue.Field(i).Interface()
		field := messValue.FieldByName(messType.Field(i).Name)

		switch inter.(type) {
		case string:
			tmp := messValue.FieldByName(messType.Field(i).Name + "Len")
			// 全转为uint64
			var stringLen uint64
			switch stringLenTmp := tmp.Interface().(type) {
			case uint16:
				stringLen = uint64(stringLenTmp)
			case uint32:
				stringLen = uint64(stringLenTmp)
			case uint64:
				stringLen = stringLenTmp
			}
			field.SetString(string(dataBuf[ptr : ptr+stringLen]))
			ptr += stringLen
		case uint16:
			field.SetUint(uint64(binary.BigEndian.Uint16(dataBuf[ptr : ptr+2])))
			ptr += 2
		case uint32:
			field.SetUint(uint64(binary.BigEndian.Uint32(dataBuf[ptr : ptr+4])))
			ptr += 4
		case uint64:
			field.SetUint(uint64(binary.BigEndian.Uint64(dataBuf[ptr : ptr+8])))
			ptr += 8
		case []byte:
			tmp := messValue.FieldByName(messType.Field(i).Name + "Len")
			var byteLen uint64
			switch byteLenTmp := tmp.Interface().(type) {
			case uint16:
				byteLen = uint64(byteLenTmp)
			case uint32:
				byteLen = uint64(byteLenTmp)
			case uint64:
				byteLen = byteLenTmp
			}
			field.SetBytes(dataBuf[ptr : ptr+byteLen])
			ptr += byteLen
		default:
			return header, nil, errors.New("unknown error")
		}
	}

	return header, mess, nil
}

func (message *RawMessage) DeconstructSuffix() {}

func (message *RawMessage) SendMessage() {
	finalBuffer := append(message.HeaderBuffer, message.DataBuffer...)
	message.Conn.Write(finalBuffer)
	// Don't forget to set both Buffer to nil!!!
	message.HeaderBuffer = nil
	message.DataBuffer = nil
}
