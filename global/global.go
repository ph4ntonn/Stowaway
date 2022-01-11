package global

import (
	"Stowaway/protocol"
	"net"
)

var G_Component *protocol.MessageComponent

func InitialGComponent(conn net.Conn, secret, uuid, token string) {
	G_Component = &protocol.MessageComponent{
		Secret: secret,
		Conn:   conn,
		UUID:   uuid,
		Token:  token,
	}
}

func UpdateGComponent(conn net.Conn) {
	G_Component.Conn = conn
}
