package global

import (
	"net"

	"Stowaway/protocol"
)

var G_TLSEnable bool
var G_Component *protocol.MessageComponent

func InitialGComponent(conn net.Conn, secret, uuid string) {
	G_Component = &protocol.MessageComponent{
		Secret: secret,
		Conn:   conn,
		UUID:   uuid,
	}
}

func UpdateGComponent(conn net.Conn) {
	G_Component.Conn = conn
}
