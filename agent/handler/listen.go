/*
 * @Author: ph4ntom
 * @Date: 2021-03-18 18:33:57
 * @LastEditors: ph4ntom
 * @LastEditTime: 2021-03-18 18:55:47
 */
package handler

// import (
// 	"Stowaway/protocol"
// 	"Stowaway/share"
// 	"log"
// 	"net"
// )

// func StartListen(addr string) {
// 	var sMessage, rMessage protocol.Message

// 	hiMess := protocol.HIMess{
// 		GreetingLen: uint16(len("Keep slient")),
// 		Greeting:    "Keep slient",
// 		IsAdmin:     0,
// 	}

// 	header := protocol.Header{
// 		Sender:      protocol.TEMP_UUID,
// 		Accepter:    protocol.ADMIN_UUID,
// 		MessageType: protocol.HI,
// 		RouteLen:    uint32(len([]byte(protocol.TEMP_ROUTE))),
// 		Route:       protocol.TEMP_ROUTE,
// 	}

// 	listener, err := net.Listen("tcp", addr)
// 	if err != nil {
// 		log.Fatalf("[*]Error occured: %s", err.Error())
// 	}

// 	defer func() {
// 		listener.Close()
// 	}()

// 	for {
// 		conn, err := listener.Accept()
// 		if err != nil {
// 			log.Printf("[*]Error occured: %s\n", err.Error())
// 			conn.Close()
// 			continue
// 		}

// 		if err := share.PassivePreAuth(conn, userOptions.Secret); err != nil {
// 			log.Fatalf("[*]Error occured: %s", err.Error())
// 		}

// 		rMessage = protocol.PrepareAndDecideWhichRProto(conn, userOptions.Secret, protocol.TEMP_UUID)
// 		fHeader, fMessage, err := protocol.DestructMessage(rMessage)

// 		if err != nil {
// 			log.Printf("[*]Fail to set connection from %s, Error: %s\n", conn.RemoteAddr().String(), err.Error())
// 			conn.Close()
// 			continue
// 		}

// 		if fHeader.MessageType == protocol.HI {
// 			mmess := fMessage.(*protocol.HIMess)
// 			if mmess.Greeting == "Shhh..." && mmess.IsAdmin == 1 {
// 				sMessage = protocol.PrepareAndDecideWhichSProto(conn, userOptions.Secret, protocol.TEMP_UUID)
// 				protocol.ConstructMessage(sMessage, header, hiMess)
// 				sMessage.SendMessage()
// 				uuid := achieveUUID(conn, userOptions.Secret)
// 				log.Printf("[*]Connection from admin %s is set up successfully!\n", conn.RemoteAddr().String())
// 				return conn, uuid
// 			}
// 		}

// 		conn.Close()
// 		log.Println("[*]Incoming connection seems illegal!")
// 	}
// }
