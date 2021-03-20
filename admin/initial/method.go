/*
 * @Author: ph4ntom
 * @Date: 2021-03-08 18:03:48
 * @LastEditors: ph4ntom
 * @LastEditTime: 2021-03-20 15:55:08
 */
package initial

import (
	"Stowaway/admin/topology"
	"Stowaway/protocol"
	"Stowaway/share"
	"Stowaway/utils"
	"log"
	"net"
)

func dispatchUUID(conn net.Conn, secret string) string {
	var sMessage, rMessage protocol.Message

	uuid := utils.GenerateUUID()
	uuidMess := protocol.UUIDMess{
		UUIDLen: uint16(len(uuid)),
		UUID:    uuid,
	}

	header := protocol.Header{
		Sender:      protocol.ADMIN_UUID,
		Accepter:    protocol.TEMP_UUID,
		MessageType: protocol.UUID,
		RouteLen:    uint32(len([]byte(protocol.TEMP_ROUTE))),
		Route:       protocol.TEMP_ROUTE,
	}

	sMessage = protocol.PrepareAndDecideWhichSProto(conn, secret, protocol.ADMIN_UUID)

	protocol.ConstructMessage(sMessage, header, uuidMess)
	sMessage.SendMessage()

	rMessage = protocol.PrepareAndDecideWhichRProto(conn, secret, protocol.ADMIN_UUID)
	fHeader, fMessage, err := protocol.DestructMessage(rMessage)

	if err != nil {
		conn.Close()
		log.Fatalf("[*]Fail to dispatch UUID to node %s, Error: %s", conn.RemoteAddr().String(), err.Error())
	}

	if fHeader.MessageType == protocol.UUIDRET {
		mmess := fMessage.(*protocol.UUIDRetMess)
		if mmess.OK != 1 {
			log.Fatalf("[*]Fail to dispatch UUID to node %s, Error: %s", conn.RemoteAddr().String(), err.Error())
		}
	}

	return uuid
}

/**
 * @description: Connect to node actively
 * @param {*Options} userOptions
 * @return {*}
 */
func NormalActive(userOptions *Options, topo *topology.Topology) net.Conn {

	var sMessage, rMessage protocol.Message

	hiMess := protocol.HIMess{
		GreetingLen: uint16(len("Shhh...")),
		Greeting:    "Shhh...",
		IsAdmin:     1,
	}

	header := protocol.Header{
		Sender:      protocol.ADMIN_UUID,
		Accepter:    protocol.TEMP_UUID,
		MessageType: protocol.HI,
		RouteLen:    uint32(len([]byte(protocol.TEMP_ROUTE))),
		Route:       protocol.TEMP_ROUTE,
	}

	for {
		conn, err := net.Dial("tcp", userOptions.Connect)
		if err != nil {
			log.Fatal("[*]Connection refused!\n")
		}

		if err := share.ActivePreAuth(conn, userOptions.Secret); err != nil {
			log.Fatalf("[*]Error occured: %s", err.Error())
		}

		sMessage = protocol.PrepareAndDecideWhichSProto(conn, userOptions.Secret, protocol.ADMIN_UUID)

		protocol.ConstructMessage(sMessage, header, hiMess)
		sMessage.SendMessage()

		rMessage = protocol.PrepareAndDecideWhichRProto(conn, userOptions.Secret, protocol.ADMIN_UUID)
		fHeader, fMessage, err := protocol.DestructMessage(rMessage)

		if err != nil {
			conn.Close()
			log.Fatalf("[*]Fail to connect node %s, Error: %s", conn.RemoteAddr().String(), err.Error())
		}

		if fHeader.MessageType == protocol.HI {
			mmess := fMessage.(*protocol.HIMess)
			if mmess.Greeting == "Keep slient" {
				node := topology.NewNode(dispatchUUID(conn, userOptions.Secret), conn.RemoteAddr().String())
				task := &topology.TopoTask{
					Mode:    topology.ADDNODE,
					Target:  node,
					ID:      protocol.TEMP_UUID,
					IsFirst: true,
				}
				topo.TaskChan <- task

				<-topo.ResultChan

				log.Printf("[*]Connect to node %s successfully!\n", conn.RemoteAddr().String())
				return conn
			}
		}

		conn.Close()
		log.Fatal("[*]Target node seems illegal!\n")
	}
}

func NormalPassive(userOptions *Options, topo *topology.Topology) net.Conn {
	listenAddr, _, err := utils.CheckIPPort(userOptions.Listen)
	if err != nil {
		log.Fatalf("[*]Error occured: %s", err.Error())
	}

	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Fatalf("[*]Error occured: %s", err.Error())
	}

	defer func() {
		listener.Close() // don't forget close the listener
	}()

	var sMessage, rMessage protocol.Message

	// just say hi!
	hiMess := protocol.HIMess{
		GreetingLen: uint16(len("Keep slient")),
		Greeting:    "Keep slient",
		IsAdmin:     1,
	}

	header := protocol.Header{
		Sender:      protocol.ADMIN_UUID,
		Accepter:    protocol.TEMP_UUID,
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

		rMessage = protocol.PrepareAndDecideWhichRProto(conn, userOptions.Secret, protocol.ADMIN_UUID)
		fHeader, fMessage, err := protocol.DestructMessage(rMessage)

		if err != nil {
			log.Printf("[*]Fail to set connection from %s, Error: %s\n", conn.RemoteAddr().String(), err.Error())
			conn.Close()
			continue
		}

		if fHeader.MessageType == protocol.HI {
			mmess := fMessage.(*protocol.HIMess)
			if mmess.Greeting == "Shhh..." {
				sMessage = protocol.PrepareAndDecideWhichSProto(conn, userOptions.Secret, protocol.ADMIN_UUID)
				protocol.ConstructMessage(sMessage, header, hiMess)
				sMessage.SendMessage()
				node := topology.NewNode(dispatchUUID(conn, userOptions.Secret), conn.RemoteAddr().String())
				task := &topology.TopoTask{
					Mode:    topology.ADDNODE,
					Target:  node,
					ID:      protocol.TEMP_UUID,
					IsFirst: true,
				}
				topo.TaskChan <- task

				<-topo.ResultChan

				log.Printf("[*]Connection from node %s is set up successfully!\n", conn.RemoteAddr().String())
				return conn
			}
		}

		conn.Close()
		log.Println("[*]Incoming connection seems illegal!")
	}
}
