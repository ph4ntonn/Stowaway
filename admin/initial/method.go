/*
 * @Author: ph4ntom
 * @Date: 2021-03-08 18:03:48
 * @LastEditors: ph4ntom
 * @LastEditTime: 2021-03-26 16:50:20
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

	sMessage = protocol.PrepareAndDecideWhichSProtoToLower(conn, secret, protocol.ADMIN_UUID)

	protocol.ConstructMessage(sMessage, header, uuidMess, false)
	sMessage.SendMessage()

	return uuid
}

/**
 * @description: Connect to node actively
 * @param {*Options} userOptions
 * @return {*}
 */
func NormalActive(userOptions *Options, topo *topology.Topology, proxy *share.Proxy) net.Conn {

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
			log.Fatalf("[*]Error occured: %s", err.Error())
		}

		if err := share.ActivePreAuth(conn, userOptions.Secret); err != nil {
			log.Fatalf("[*]Error occured: %s", err.Error())
		}

		sMessage = protocol.PrepareAndDecideWhichSProtoToLower(conn, userOptions.Secret, protocol.ADMIN_UUID)

		protocol.ConstructMessage(sMessage, header, hiMess, false)
		sMessage.SendMessage()

		rMessage = protocol.PrepareAndDecideWhichRProtoFromLower(conn, userOptions.Secret, protocol.ADMIN_UUID)
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
					UUID:    protocol.TEMP_UUID,
					IsFirst: true,
				}
				topo.TaskChan <- task

				<-topo.ResultChan

				log.Printf("[*]Connect to node %s successfully! Node id is 0\n", conn.RemoteAddr().String())
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
			log.Printf("[*]Error occured: %s\n", err.Error())
			conn.Close()
			continue
		}

		if err := share.PassivePreAuth(conn, userOptions.Secret); err != nil {
			log.Fatalf("[*]Error occured: %s", err.Error())
		}

		rMessage = protocol.PrepareAndDecideWhichRProtoFromLower(conn, userOptions.Secret, protocol.ADMIN_UUID)
		fHeader, fMessage, err := protocol.DestructMessage(rMessage)

		if err != nil {
			log.Printf("[*]Fail to set connection from %s, Error: %s\n", conn.RemoteAddr().String(), err.Error())
			conn.Close()
			continue
		}

		if fHeader.MessageType == protocol.HI {
			mmess := fMessage.(*protocol.HIMess)
			if mmess.Greeting == "Shhh..." && mmess.IsAdmin == 0 {
				sMessage = protocol.PrepareAndDecideWhichSProtoToLower(conn, userOptions.Secret, protocol.ADMIN_UUID)
				protocol.ConstructMessage(sMessage, header, hiMess, false)
				sMessage.SendMessage()
				if mmess.IsReconnect == 0 {
					node := topology.NewNode(dispatchUUID(conn, userOptions.Secret), conn.RemoteAddr().String())
					task := &topology.TopoTask{
						Mode:    topology.ADDNODE,
						Target:  node,
						UUID:    protocol.TEMP_UUID,
						IsFirst: true,
					}
					topo.TaskChan <- task

					<-topo.ResultChan

					log.Printf("[*]Connection from node %s is set up successfully! Node id is 0\n", conn.RemoteAddr().String())
				} else {

				}

				return conn
			}
		}

		conn.Close()
		log.Println("[*]Incoming connection seems illegal!")
	}
}
