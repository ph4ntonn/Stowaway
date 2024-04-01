package handler

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"time"

	"Stowaway/agent/initial"
	"Stowaway/agent/manager"
	"Stowaway/global"
	"Stowaway/protocol"
	"Stowaway/share"
	"Stowaway/share/transport"

	reuseport "github.com/libp2p/go-reuseport"
)

const (
	NORMAL = iota
	IPTABLES
	SOREUSE
)

type Listen struct {
	method int
	addr   string
}

func newListen(method int, addr string) *Listen {
	listen := new(Listen)
	listen.method = method
	listen.addr = addr
	return listen
}

func (listen *Listen) start(mgr *manager.Manager, options *initial.Options) {
	sUMessage := protocol.NewUpMsg(global.G_Component.Conn, global.G_Component.Secret, global.G_Component.UUID)

	resHeader := &protocol.Header{
		Sender:      global.G_Component.UUID,
		Accepter:    protocol.ADMIN_UUID,
		MessageType: protocol.LISTENRES,
		RouteLen:    uint32(len([]byte(protocol.TEMP_ROUTE))), // No need to set route when agent send mess to admin
		Route:       protocol.TEMP_ROUTE,
	}

	failMess := &protocol.ListenRes{
		OK: 0,
	}

	if listen.method == IPTABLES {
		if options.ReusePort == "" { // means node is not initialed via iptable reuse
			protocol.ConstructMessage(sUMessage, resHeader, failMess, false)
			sUMessage.SendMessage()
			return
		}
	} else if listen.method == SOREUSE {
		if options.ReuseHost == "" {
			protocol.ConstructMessage(sUMessage, resHeader, failMess, false)
			sUMessage.SendMessage()
			return
		}
	}

	switch listen.method {
	case NORMAL:
		go listen.normalListen(mgr, options)
	case IPTABLES:
		go listen.iptablesListen(mgr, options)
	case SOREUSE:
		go listen.soReuseListen(mgr, options)
	}

}

func (listen *Listen) normalListen(mgr *manager.Manager, options *initial.Options) {
	sUMessage := protocol.NewUpMsg(global.G_Component.Conn, global.G_Component.Secret, global.G_Component.UUID)

	resHeader := &protocol.Header{
		Sender:      global.G_Component.UUID,
		Accepter:    protocol.ADMIN_UUID,
		MessageType: protocol.LISTENRES,
		RouteLen:    uint32(len([]byte(protocol.TEMP_ROUTE))), // No need to set route when agent send mess to admin
		Route:       protocol.TEMP_ROUTE,
	}

	failMess := &protocol.ListenRes{
		OK: 0,
	}

	succMess := &protocol.ListenRes{
		OK: 1,
	}

	listener, err := net.Listen("tcp", listen.addr)
	if err != nil {
		protocol.ConstructMessage(sUMessage, resHeader, failMess, false)
		sUMessage.SendMessage()
		return
	}

	defer listener.Close()

	protocol.ConstructMessage(sUMessage, resHeader, succMess, false)
	sUMessage.SendMessage()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("[*] Error occurred: %s\n", err.Error())
			continue
		}

		if global.G_TLSEnable {
			var tlsConfig *tls.Config
			tlsConfig, err = transport.NewServerTLSConfig()
			if err != nil {
				log.Printf("[*] Error occured: %s", err.Error())
				conn.Close()
				continue
			}
			conn = transport.WrapTLSServerConn(conn, tlsConfig)
		}

		param := new(protocol.NegParam)
		param.Conn = conn
		proto := protocol.NewDownProto(param)
		proto.SNegotiate()

		if err := share.PassivePreAuth(conn); err != nil {
			conn.Close()
			continue
		}

		rMessage := protocol.NewDownMsg(conn, global.G_Component.Secret, protocol.ADMIN_UUID) //fake admin
		fHeader, fMessage, err := protocol.DestructMessage(rMessage)
		if err != nil {
			conn.Close()
			continue
		}

		if fHeader.MessageType == protocol.HI {
			mmess := fMessage.(*protocol.HIMess)

			if mmess.Greeting == "Shhh..." && mmess.IsAdmin == 0 {
				var childUUID string

				sLMessage := protocol.NewDownMsg(conn, global.G_Component.Secret, protocol.ADMIN_UUID) //fake admin

				hiMess := &protocol.HIMess{
					GreetingLen: uint16(len("Keep slient")),
					Greeting:    "Keep slient",
					UUIDLen:     uint16(len(protocol.ADMIN_UUID)),
					UUID:        protocol.ADMIN_UUID,
					IsAdmin:     1,
					IsReconnect: 0,
				}

				hiHeader := &protocol.Header{
					Sender:      protocol.ADMIN_UUID,
					Accepter:    protocol.TEMP_UUID,
					MessageType: protocol.HI,
					RouteLen:    uint32(len([]byte(protocol.TEMP_ROUTE))),
					Route:       protocol.TEMP_ROUTE,
				}

				protocol.ConstructMessage(sLMessage, hiHeader, hiMess, false)
				sLMessage.SendMessage()

				if mmess.IsReconnect == 0 {
					childIP := conn.RemoteAddr().String()

					cUUIDReqHeader := &protocol.Header{
						Sender:      global.G_Component.UUID,
						Accepter:    protocol.ADMIN_UUID,
						MessageType: protocol.CHILDUUIDREQ,
						RouteLen:    uint32(len([]byte(protocol.TEMP_ROUTE))),
						Route:       protocol.TEMP_ROUTE,
					}

					cUUIDMess := &protocol.ChildUUIDReq{
						ParentUUIDLen: uint16(len(global.G_Component.UUID)),
						ParentUUID:    global.G_Component.UUID,
						IPLen:         uint16(len(childIP)),
						IP:            childIP,
					}

					protocol.ConstructMessage(sUMessage, cUUIDReqHeader, cUUIDMess, false)
					sUMessage.SendMessage()

					// Problem:If two listen port ready,and at the same time,two agent come,then maybe it would cause wrong uuid dispatching,but i think such coincidence hardly happen
					// (Fine..I just don't want to maintain status Orz
					childUUID = <-mgr.ListenManager.ChildUUIDChan

					uuidHeader := &protocol.Header{
						Sender:      protocol.ADMIN_UUID, // Fake admin LOL
						Accepter:    protocol.TEMP_UUID,
						MessageType: protocol.UUID,
						RouteLen:    uint32(len([]byte(protocol.TEMP_ROUTE))),
						Route:       protocol.TEMP_ROUTE,
					}

					uuidMess := &protocol.UUIDMess{
						UUIDLen: uint16(len(childUUID)),
						UUID:    childUUID,
					}

					protocol.ConstructMessage(sLMessage, uuidHeader, uuidMess, false)
					sLMessage.SendMessage()
				} else {
					reheader := &protocol.Header{
						Sender:      global.G_Component.UUID,
						Accepter:    protocol.ADMIN_UUID,
						MessageType: protocol.NODEREONLINE,
						RouteLen:    uint32(len([]byte(protocol.TEMP_ROUTE))),
						Route:       protocol.TEMP_ROUTE,
					}

					reMess := &protocol.NodeReonline{
						ParentUUIDLen: uint16(len(global.G_Component.UUID)),
						ParentUUID:    global.G_Component.UUID,
						UUIDLen:       uint16(len(mmess.UUID)),
						UUID:          mmess.UUID,
						IPLen:         uint16(len(conn.RemoteAddr().String())),
						IP:            conn.RemoteAddr().String(),
					}

					protocol.ConstructMessage(sUMessage, reheader, reMess, false)
					sUMessage.SendMessage()

					childUUID = mmess.UUID
				}

				childrenTask := &manager.ChildrenTask{
					Mode: manager.C_NEWCHILD,
					UUID: childUUID,
					Conn: conn,
				}
				mgr.ChildrenManager.TaskChan <- childrenTask
				<-mgr.ChildrenManager.ResultChan

				mgr.ChildrenManager.ChildComeChan <- &manager.ChildInfo{UUID: childUUID, Conn: conn}

				return
			}
		}

		conn.Close()
	}
}

func (listen *Listen) iptablesListen(mgr *manager.Manager, options *initial.Options) {
	sUMessage := protocol.NewUpMsg(global.G_Component.Conn, global.G_Component.Secret, global.G_Component.UUID)

	resHeader := &protocol.Header{
		Sender:      global.G_Component.UUID,
		Accepter:    protocol.ADMIN_UUID,
		MessageType: protocol.LISTENRES,
		RouteLen:    uint32(len([]byte(protocol.TEMP_ROUTE))), // No need to set route when agent send mess to admin
		Route:       protocol.TEMP_ROUTE,
	}

	failMess := &protocol.ListenRes{
		OK: 0,
	}

	succMess := &protocol.ListenRes{
		OK: 1,
	}

	// try set rules again
	initial.SetPortReuseRules(options.Listen, options.ReusePort)

	listenAddr := fmt.Sprintf("0.0.0.0:%s", options.Listen)
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		protocol.ConstructMessage(sUMessage, resHeader, failMess, false)
		sUMessage.SendMessage()
		return
	}

	defer listener.Close()

	protocol.ConstructMessage(sUMessage, resHeader, succMess, false)
	sUMessage.SendMessage()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("[*] Error occurred: %s\n", err.Error())
			continue
		}

		if global.G_TLSEnable {
			var tlsConfig *tls.Config
			tlsConfig, err = transport.NewServerTLSConfig()
			if err != nil {
				log.Printf("[*] Error occured: %s", err.Error())
				conn.Close()
				continue
			}
			conn = transport.WrapTLSServerConn(conn, tlsConfig)
		}

		param := new(protocol.NegParam)
		param.Conn = conn
		proto := protocol.NewDownProto(param)
		proto.SNegotiate()

		if err := share.PassivePreAuth(conn); err != nil {
			conn.Close()
			continue
		}

		rMessage := protocol.NewDownMsg(conn, global.G_Component.Secret, protocol.ADMIN_UUID) //fake admin
		fHeader, fMessage, err := protocol.DestructMessage(rMessage)

		if err != nil {
			conn.Close()
			continue
		}

		if fHeader.MessageType == protocol.HI {
			mmess := fMessage.(*protocol.HIMess)

			if mmess.Greeting == "Shhh..." && mmess.IsAdmin == 0 {
				var childUUID string

				sLMessage := protocol.NewDownMsg(conn, global.G_Component.Secret, protocol.ADMIN_UUID) //fake admin

				hiMess := &protocol.HIMess{
					GreetingLen: uint16(len("Keep slient")),
					Greeting:    "Keep slient",
					UUIDLen:     uint16(len(protocol.ADMIN_UUID)),
					UUID:        protocol.ADMIN_UUID,
					IsAdmin:     1,
					IsReconnect: 0,
				}

				hiHeader := &protocol.Header{
					Sender:      protocol.ADMIN_UUID,
					Accepter:    protocol.TEMP_UUID,
					MessageType: protocol.HI,
					RouteLen:    uint32(len([]byte(protocol.TEMP_ROUTE))),
					Route:       protocol.TEMP_ROUTE,
				}

				protocol.ConstructMessage(sLMessage, hiHeader, hiMess, false)
				sLMessage.SendMessage()

				if mmess.IsReconnect == 0 {
					childIP := conn.RemoteAddr().String()

					cUUIDReqHeader := &protocol.Header{
						Sender:      global.G_Component.UUID,
						Accepter:    protocol.ADMIN_UUID,
						MessageType: protocol.CHILDUUIDREQ,
						RouteLen:    uint32(len([]byte(protocol.TEMP_ROUTE))),
						Route:       protocol.TEMP_ROUTE,
					}

					cUUIDMess := &protocol.ChildUUIDReq{
						ParentUUIDLen: uint16(len(global.G_Component.UUID)),
						ParentUUID:    global.G_Component.UUID,
						IPLen:         uint16(len(childIP)),
						IP:            childIP,
					}

					protocol.ConstructMessage(sUMessage, cUUIDReqHeader, cUUIDMess, false)
					sUMessage.SendMessage()

					// Problem:If two listen port ready,and at the same time,two agent come,then maybe it would cause wrong uuid dispatching,but i think such coincidence hardly happen
					// (Fine..I just don't want to maintain status Orz
					childUUID = <-mgr.ListenManager.ChildUUIDChan

					uuidHeader := &protocol.Header{
						Sender:      protocol.ADMIN_UUID, // Fake admin LOL
						Accepter:    protocol.TEMP_UUID,
						MessageType: protocol.UUID,
						RouteLen:    uint32(len([]byte(protocol.TEMP_ROUTE))),
						Route:       protocol.TEMP_ROUTE,
					}

					uuidMess := &protocol.UUIDMess{
						UUIDLen: uint16(len(childUUID)),
						UUID:    childUUID,
					}

					protocol.ConstructMessage(sLMessage, uuidHeader, uuidMess, false)
					sLMessage.SendMessage()
				} else {
					reheader := &protocol.Header{
						Sender:      global.G_Component.UUID,
						Accepter:    protocol.ADMIN_UUID,
						MessageType: protocol.NODEREONLINE,
						RouteLen:    uint32(len([]byte(protocol.TEMP_ROUTE))),
						Route:       protocol.TEMP_ROUTE,
					}

					reMess := &protocol.NodeReonline{
						ParentUUIDLen: uint16(len(global.G_Component.UUID)),
						ParentUUID:    global.G_Component.UUID,
						UUIDLen:       uint16(len(mmess.UUID)),
						UUID:          mmess.UUID,
						IPLen:         uint16(len(conn.RemoteAddr().String())),
						IP:            conn.RemoteAddr().String(),
					}

					protocol.ConstructMessage(sUMessage, reheader, reMess, false)
					sUMessage.SendMessage()

					childUUID = mmess.UUID
				}

				childrenTask := &manager.ChildrenTask{
					Mode: manager.C_NEWCHILD,
					UUID: childUUID,
					Conn: conn,
				}
				mgr.ChildrenManager.TaskChan <- childrenTask
				<-mgr.ChildrenManager.ResultChan

				mgr.ChildrenManager.ChildComeChan <- &manager.ChildInfo{UUID: childUUID, Conn: conn}

				return
			}
		}

		conn.Close()
	}
}

func (listen *Listen) soReuseListen(mgr *manager.Manager, options *initial.Options) {
	sUMessage := protocol.NewUpMsg(global.G_Component.Conn, global.G_Component.Secret, global.G_Component.UUID)

	resHeader := &protocol.Header{
		Sender:      global.G_Component.UUID,
		Accepter:    protocol.ADMIN_UUID,
		MessageType: protocol.LISTENRES,
		RouteLen:    uint32(len([]byte(protocol.TEMP_ROUTE))), // No need to set route when agent send mess to admin
		Route:       protocol.TEMP_ROUTE,
	}

	failMess := &protocol.ListenRes{
		OK: 0,
	}

	succMess := &protocol.ListenRes{
		OK: 1,
	}

	listenAddr := fmt.Sprintf("%s:%s", options.ReuseHost, options.ReusePort)
	listener, err := reuseport.Listen("tcp", listenAddr)
	if err != nil {
		protocol.ConstructMessage(sUMessage, resHeader, failMess, false)
		sUMessage.SendMessage()
		return
	}

	defer listener.Close()

	protocol.ConstructMessage(sUMessage, resHeader, succMess, false)
	sUMessage.SendMessage()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("[*] Error occurred: %s\n", err.Error())
			continue
		}

		if global.G_TLSEnable {
			var tlsConfig *tls.Config
			tlsConfig, err = transport.NewServerTLSConfig()
			if err != nil {
				log.Printf("[*] Error occured: %s", err.Error())
				conn.Close()
				continue
			}
			conn = transport.WrapTLSServerConn(conn, tlsConfig)
		}

		param := new(protocol.NegParam)
		param.Conn = conn
		proto := protocol.NewDownProto(param)
		proto.SNegotiate()

		defer conn.SetReadDeadline(time.Time{})
		conn.SetReadDeadline(time.Now().Add(2 * time.Second))

		buffer := make([]byte, 16)
		count, err := io.ReadFull(conn, buffer)

		if err != nil {
			if timeoutErr, ok := err.(net.Error); ok && timeoutErr.Timeout() {
				go initial.ProxyStream(conn, buffer[:count], options.ReusePort)
				continue
			} else {
				conn.Close()
				continue
			}
		}

		if string(buffer[:count]) == share.AuthToken {
			conn.Write([]byte(share.AuthToken))
		} else {
			go initial.ProxyStream(conn, buffer[:count], options.ReusePort)
			continue
		}

		rMessage := protocol.NewDownMsg(conn, global.G_Component.Secret, protocol.ADMIN_UUID) //fake admin
		fHeader, fMessage, err := protocol.DestructMessage(rMessage)

		if err != nil {
			conn.Close()
			continue
		}

		if fHeader.MessageType == protocol.HI {
			mmess := fMessage.(*protocol.HIMess)

			if mmess.Greeting == "Shhh..." && mmess.IsAdmin == 0 {
				var childUUID string

				sLMessage := protocol.NewDownMsg(conn, global.G_Component.Secret, protocol.ADMIN_UUID) //fake admin

				hiMess := &protocol.HIMess{
					GreetingLen: uint16(len("Keep slient")),
					Greeting:    "Keep slient",
					UUIDLen:     uint16(len(protocol.ADMIN_UUID)),
					UUID:        protocol.ADMIN_UUID,
					IsAdmin:     1,
					IsReconnect: 0,
				}

				hiHeader := &protocol.Header{
					Sender:      protocol.ADMIN_UUID,
					Accepter:    protocol.TEMP_UUID,
					MessageType: protocol.HI,
					RouteLen:    uint32(len([]byte(protocol.TEMP_ROUTE))),
					Route:       protocol.TEMP_ROUTE,
				}

				protocol.ConstructMessage(sLMessage, hiHeader, hiMess, false)
				sLMessage.SendMessage()

				if mmess.IsReconnect == 0 {
					childIP := conn.RemoteAddr().String()

					cUUIDReqHeader := &protocol.Header{
						Sender:      global.G_Component.UUID,
						Accepter:    protocol.ADMIN_UUID,
						MessageType: protocol.CHILDUUIDREQ,
						RouteLen:    uint32(len([]byte(protocol.TEMP_ROUTE))),
						Route:       protocol.TEMP_ROUTE,
					}

					cUUIDMess := &protocol.ChildUUIDReq{
						ParentUUIDLen: uint16(len(global.G_Component.UUID)),
						ParentUUID:    global.G_Component.UUID,
						IPLen:         uint16(len(childIP)),
						IP:            childIP,
					}

					protocol.ConstructMessage(sUMessage, cUUIDReqHeader, cUUIDMess, false)
					sUMessage.SendMessage()

					// Problem:If two listen port ready,and at the same time,two agent come,then maybe it would cause wrong uuid dispatching,but i think such coincidence hardly happen
					// (Fine..I just don't want to maintain status Orz
					childUUID = <-mgr.ListenManager.ChildUUIDChan

					uuidHeader := &protocol.Header{
						Sender:      protocol.ADMIN_UUID, // Fake admin LOL
						Accepter:    protocol.TEMP_UUID,
						MessageType: protocol.UUID,
						RouteLen:    uint32(len([]byte(protocol.TEMP_ROUTE))),
						Route:       protocol.TEMP_ROUTE,
					}

					uuidMess := &protocol.UUIDMess{
						UUIDLen: uint16(len(childUUID)),
						UUID:    childUUID,
					}

					protocol.ConstructMessage(sLMessage, uuidHeader, uuidMess, false)
					sLMessage.SendMessage()
				} else {
					reheader := &protocol.Header{
						Sender:      global.G_Component.UUID,
						Accepter:    protocol.ADMIN_UUID,
						MessageType: protocol.NODEREONLINE,
						RouteLen:    uint32(len([]byte(protocol.TEMP_ROUTE))),
						Route:       protocol.TEMP_ROUTE,
					}

					reMess := &protocol.NodeReonline{
						ParentUUIDLen: uint16(len(global.G_Component.UUID)),
						ParentUUID:    global.G_Component.UUID,
						UUIDLen:       uint16(len(mmess.UUID)),
						UUID:          mmess.UUID,
						IPLen:         uint16(len(conn.RemoteAddr().String())),
						IP:            conn.RemoteAddr().String(),
					}

					protocol.ConstructMessage(sUMessage, reheader, reMess, false)
					sUMessage.SendMessage()

					childUUID = mmess.UUID
				}

				childrenTask := &manager.ChildrenTask{
					Mode: manager.C_NEWCHILD,
					UUID: childUUID,
					Conn: conn,
				}
				mgr.ChildrenManager.TaskChan <- childrenTask
				<-mgr.ChildrenManager.ResultChan

				mgr.ChildrenManager.ChildComeChan <- &manager.ChildInfo{UUID: childUUID, Conn: conn}

				return
			}
		}

		conn.Close()
	}
}

func DispatchListenMess(mgr *manager.Manager, options *initial.Options) {
	for {
		message := <-mgr.ListenManager.ListenMessChan

		switch mess := message.(type) {
		case *protocol.ListenReq:
			listen := newListen(int(mess.Method), mess.Addr)
			go listen.start(mgr, options)
		case *protocol.ChildUUIDRes:
			mgr.ListenManager.ChildUUIDChan <- mess.UUID
		}
	}
}
