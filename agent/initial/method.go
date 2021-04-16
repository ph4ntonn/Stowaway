/*
 * @Author: ph4ntom
 * @Date: 2021-03-10 15:28:20
 * @LastEditors: ph4ntom
 * @LastEditTime: 2021-03-26 16:53:34
 */
package initial

import (
	"Stowaway/protocol"
	"Stowaway/share"
	"Stowaway/utils"
	"log"
	"net"
)

func achieveUUID(conn net.Conn, secret string) (uuid string) {
	var rMessage protocol.Message

	rMessage = protocol.PrepareAndDecideWhichRProtoFromUpper(conn, secret, protocol.TEMP_UUID)
	fHeader, fMessage, err := protocol.DestructMessage(rMessage)

	if err != nil {
		conn.Close()
		log.Fatalf("[*]Fail to achieve UUID from admin %s, Error: %s", conn.RemoteAddr().String(), err.Error())
	}

	if fHeader.MessageType == protocol.UUID {
		mmess := fMessage.(*protocol.UUIDMess)
		uuid = mmess.UUID
	}

	return uuid
}

func NormalActive(userOptions *Options, proxy *share.Proxy) (net.Conn, string) {
	var sMessage, rMessage protocol.Message
	// just say hi!
	hiMess := &protocol.HIMess{
		GreetingLen: uint16(len("Shhh...")),
		Greeting:    "Shhh...",
		IsAdmin:     0,
	}

	header := &protocol.Header{
		Sender:      protocol.TEMP_UUID,
		Accepter:    protocol.ADMIN_UUID,
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
			log.Fatalf("[*]Error occured: %s", err.Error())
		}

		if err := share.ActivePreAuth(conn, userOptions.Secret); err != nil {
			log.Fatalf("[*]Error occured: %s", err.Error())
		}

		sMessage = protocol.PrepareAndDecideWhichSProtoToUpper(conn, userOptions.Secret, protocol.TEMP_UUID)

		protocol.ConstructMessage(sMessage, header, hiMess, false)
		sMessage.SendMessage()

		rMessage = protocol.PrepareAndDecideWhichRProtoFromUpper(conn, userOptions.Secret, protocol.TEMP_UUID)
		fHeader, fMessage, err := protocol.DestructMessage(rMessage)

		if err != nil {
			conn.Close()
			log.Fatalf("[*]Fail to connect admin %s, Error: %s", conn.RemoteAddr().String(), err.Error())
		}

		if fHeader.MessageType == protocol.HI {
			mmess := fMessage.(*protocol.HIMess)
			if mmess.Greeting == "Keep slient" && mmess.IsAdmin == 1 {
				uuid := achieveUUID(conn, userOptions.Secret)
				log.Printf("[*]Connect to admin %s successfully!\n", conn.RemoteAddr().String())
				return conn, uuid
			}
		}

		conn.Close()
		log.Fatal("[*]Admin seems illegal!\n")
	}
}

func NormalPassive(userOptions *Options) (net.Conn, string) {
	listenAddr, _, err := utils.CheckIPPort(userOptions.Listen)
	if err != nil {
		log.Fatalf("[*]Error occured: %s", err.Error())
	}

	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Fatalf("[*]Error occured: %s", err.Error())
	}

	defer func() {
		listener.Close()
	}()

	var sMessage, rMessage protocol.Message

	hiMess := &protocol.HIMess{
		GreetingLen: uint16(len("Keep slient")),
		Greeting:    "Keep slient",
		IsAdmin:     0,
	}

	header := &protocol.Header{
		Sender:      protocol.TEMP_UUID,
		Accepter:    protocol.ADMIN_UUID,
		MessageType: protocol.HI,
		RouteLen:    uint32(len([]byte(protocol.TEMP_ROUTE))),
		Route:       protocol.TEMP_ROUTE,
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("[*]Error occured: %s\n", err.Error())
			conn.Close()
			continue
		}

		if err := share.PassivePreAuth(conn, userOptions.Secret); err != nil {
			log.Fatalf("[*]Error occured: %s", err.Error())
		}

		rMessage = protocol.PrepareAndDecideWhichRProtoFromUpper(conn, userOptions.Secret, protocol.TEMP_UUID)
		fHeader, fMessage, err := protocol.DestructMessage(rMessage)

		if err != nil {
			log.Printf("[*]Fail to set connection from %s, Error: %s\n", conn.RemoteAddr().String(), err.Error())
			conn.Close()
			continue
		}

		if fHeader.MessageType == protocol.HI {
			mmess := fMessage.(*protocol.HIMess)
			if mmess.Greeting == "Shhh..." && mmess.IsAdmin == 1 {
				sMessage = protocol.PrepareAndDecideWhichSProtoToUpper(conn, userOptions.Secret, protocol.TEMP_UUID)
				protocol.ConstructMessage(sMessage, header, hiMess, false)
				sMessage.SendMessage()
				uuid := achieveUUID(conn, userOptions.Secret)
				log.Printf("[*]Connection from admin %s is set up successfully!\n", conn.RemoteAddr().String())
				return conn, uuid
			}
		}

		conn.Close()
		log.Println("[*]Incoming connection seems illegal!")
	}
}
