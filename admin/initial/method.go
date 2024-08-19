package initial

import (
	"crypto/tls"
	"net"
	"os"

	"Stowaway/admin/printer"
	"Stowaway/admin/topology"
	"Stowaway/protocol"
	"Stowaway/share"
	"Stowaway/share/transport"
	"Stowaway/utils"
)

func dispatchUUID(conn net.Conn, secret string) string {
	var sMessage protocol.Message

	uuid := utils.GenerateUUID()
	uuidMess := &protocol.UUIDMess{
		UUIDLen: uint16(len(uuid)),
		UUID:    uuid,
	}

	header := &protocol.Header{
		Sender:      protocol.ADMIN_UUID,
		Accepter:    protocol.TEMP_UUID,
		MessageType: protocol.UUID,
		RouteLen:    uint32(len([]byte(protocol.TEMP_ROUTE))),
		Route:       protocol.TEMP_ROUTE,
	}

	sMessage = protocol.NewDownMsg(conn, secret, protocol.ADMIN_UUID)

	protocol.ConstructMessage(sMessage, header, uuidMess, false)
	sMessage.SendMessage()

	return uuid
}

func NormalActive(userOptions *Options, topo *topology.Topology, proxy share.Proxy) net.Conn {

	var sMessage, rMessage protocol.Message

	hiMess := &protocol.HIMess{
		GreetingLen: uint16(len("Shhh...")),
		Greeting:    "Shhh...",
		UUIDLen:     uint16(len(protocol.ADMIN_UUID)),
		UUID:        protocol.ADMIN_UUID,
		IsAdmin:     1,
		IsReconnect: 0,
	}

	header := &protocol.Header{
		Sender:      protocol.ADMIN_UUID,
		Accepter:    protocol.TEMP_UUID,
		MessageType: protocol.HI,
		RouteLen:    uint32(len([]byte(protocol.TEMP_ROUTE))),
		Route:       protocol.TEMP_ROUTE,
	}

	for {
		var (
			conn net.Conn
			err  error
		)

		if proxy == nil {
			conn, err = net.Dial("tcp", userOptions.Connect)
		} else {
			conn, err = proxy.Dial()
		}

		if err != nil {
			printer.Fail("[*] Error occurred: %s", err.Error())
			os.Exit(0)
		}

		if userOptions.TlsEnable {
			var tlsConfig *tls.Config
			tlsConfig, err = transport.NewClientTLSConfig(userOptions.Domain)
			if err != nil {
				printer.Fail("[*] Error occured: %s", err.Error())
				conn.Close()
				continue
			}
			conn = transport.WrapTLSClientConn(conn, tlsConfig)
			// As we have already used TLS, we don't need to use aes inside
			// Set userOptions.Secret as null to disable aes
			userOptions.Secret = ""
		}

		param := new(protocol.NegParam)
		param.Conn = conn
		param.Domain = userOptions.Domain
		proto := protocol.NewDownProto(param)
		proto.CNegotiate()

		if err := share.ActivePreAuth(conn); err != nil {
			printer.Fail("[*] Error occurred: %s", err.Error())
			os.Exit(0)
		}

		sMessage = protocol.NewDownMsg(conn, userOptions.Secret, protocol.ADMIN_UUID)

		protocol.ConstructMessage(sMessage, header, hiMess, false)
		sMessage.SendMessage()

		rMessage = protocol.NewDownMsg(conn, userOptions.Secret, protocol.ADMIN_UUID)
		fHeader, fMessage, err := protocol.DestructMessage(rMessage)

		if err != nil {
			conn.Close()
			printer.Fail("[*] Fail to connect node %s, Error: %s", conn.RemoteAddr().String(), err.Error())
			os.Exit(0)
		}

		if fHeader.MessageType == protocol.HI {
			mmess := fMessage.(*protocol.HIMess)
			if mmess.Greeting == "Keep slient" && mmess.IsAdmin == 0 {
				if mmess.IsReconnect == 0 {
					node := topology.NewNode(dispatchUUID(conn, userOptions.Secret), conn.RemoteAddr().String())
					task := &topology.TopoTask{
						Mode:       topology.ADDNODE,
						Target:     node,
						ParentUUID: protocol.TEMP_UUID,
						IsFirst:    true,
					}
					topo.TaskChan <- task

					<-topo.ResultChan

					printer.Success("[*] Connect to node %s successfully! Node id is 0\r\n", conn.RemoteAddr().String())
					return conn
				} else {
					node := topology.NewNode(mmess.UUID, conn.RemoteAddr().String())
					task := &topology.TopoTask{
						Mode:       topology.ADDNODE,
						Target:     node,
						ParentUUID: protocol.TEMP_UUID,
						IsFirst:    true,
					}
					topo.TaskChan <- task

					<-topo.ResultChan

					printer.Success("[*] Connect to node %s successfully! Node id is 0\r\n", conn.RemoteAddr().String())
					return conn
				}
			}
		}

		conn.Close()
		printer.Fail("[*] Target node seems illegal!\n")
	}
}

func NormalPassive(userOptions *Options, topo *topology.Topology) net.Conn {
	listenAddr, _, err := utils.CheckIPPort(userOptions.Listen)
	if err != nil {
		printer.Fail("[*] Error occurred: %s", err.Error())
		os.Exit(0)
	}

	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		printer.Fail("[*] Error occurred: %s", err.Error())
		os.Exit(0)
	}

	defer func() {
		listener.Close() // don't forget close the listener
	}()

	var sMessage, rMessage protocol.Message

	// just say hi!
	hiMess := &protocol.HIMess{
		GreetingLen: uint16(len("Keep slient")),
		Greeting:    "Keep slient",
		UUIDLen:     uint16(len(protocol.ADMIN_UUID)),
		UUID:        protocol.ADMIN_UUID,
		IsAdmin:     1,
		IsReconnect: 0,
	}

	header := &protocol.Header{
		Sender:      protocol.ADMIN_UUID,
		Accepter:    protocol.TEMP_UUID,
		MessageType: protocol.HI,
		RouteLen:    uint32(len([]byte(protocol.TEMP_ROUTE))),
		Route:       protocol.TEMP_ROUTE,
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			printer.Fail("[*] Error occurred: %s\r\n", err.Error())
			continue
		}

		if userOptions.TlsEnable {
			var tlsConfig *tls.Config
			tlsConfig, err = transport.NewServerTLSConfig()
			if err != nil {
				printer.Fail("[*] Error occured: %s", err.Error())
				conn.Close()
				continue
			}
			conn = transport.WrapTLSServerConn(conn, tlsConfig)
			// As we have already used TLS, we don't need to use aes inside
			// Set userOptions.Secret as null to disable aes
			userOptions.Secret = ""
		}

		param := new(protocol.NegParam)
		param.Conn = conn
		proto := protocol.NewDownProto(param)
		proto.SNegotiate()

		if err := share.PassivePreAuth(conn); err != nil {
			printer.Fail("[*] Error occurred: %s\r\n", err.Error())
			conn.Close()
			continue
		}

		rMessage = protocol.NewDownMsg(conn, userOptions.Secret, protocol.ADMIN_UUID)
		fHeader, fMessage, err := protocol.DestructMessage(rMessage)

		if err != nil {
			printer.Fail("[*] Fail to set connection from %s, Error: %s\r\n", conn.RemoteAddr().String(), err.Error())
			conn.Close()
			continue
		}

		if fHeader.MessageType == protocol.HI {
			mmess := fMessage.(*protocol.HIMess)
			if mmess.Greeting == "Shhh..." && mmess.IsAdmin == 0 {
				sMessage = protocol.NewDownMsg(conn, userOptions.Secret, protocol.ADMIN_UUID)
				protocol.ConstructMessage(sMessage, header, hiMess, false)
				sMessage.SendMessage()

				if mmess.IsReconnect == 0 {
					node := topology.NewNode(dispatchUUID(conn, userOptions.Secret), conn.RemoteAddr().String())
					task := &topology.TopoTask{
						Mode:       topology.ADDNODE,
						Target:     node,
						ParentUUID: protocol.TEMP_UUID,
						IsFirst:    true,
					}
					topo.TaskChan <- task

					<-topo.ResultChan

					printer.Success("[*] Connection from node %s is set up successfully! Node id is 0\r\n", conn.RemoteAddr().String())
				} else {
					node := topology.NewNode(mmess.UUID, conn.RemoteAddr().String())
					task := &topology.TopoTask{
						Mode:       topology.ADDNODE,
						Target:     node,
						ParentUUID: protocol.TEMP_UUID,
						IsFirst:    true,
					}
					topo.TaskChan <- task

					<-topo.ResultChan

					printer.Success("[*] Connection from node %s is set up successfully! Node id is 0\r\n", conn.RemoteAddr().String())
				}

				return conn
			}
		}

		conn.Close()
		printer.Fail("[*] Incoming connection seems illegal!")
	}
}
