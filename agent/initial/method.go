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
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	reuseport "github.com/libp2p/go-reuseport"
)

const CHAIN_NAME = "STOWAWAY"

var START_FORWARDING string
var STOP_FORWARDING string

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

func NormalActive(userOptions *Options, proxy *share.Proxy, isReconnect uint16, uuid string) (net.Conn, string) {
	var sMessage, rMessage protocol.Message
	// just say hi!
	hiMess := &protocol.HIMess{
		GreetingLen: uint16(len("Shhh...")),
		Greeting:    "Shhh...",
		UUIDLen:     uint16(len(uuid)),
		UUID:        uuid,
		IsAdmin:     0,
		IsReconnect: isReconnect,
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

func NormalPassive(userOptions *Options, isReconnect uint16, uuid string) (net.Conn, string) {
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
		UUIDLen:     uint16(len(uuid)),
		UUID:        uuid,
		IsAdmin:     0,
		IsReconnect: isReconnect,
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

func IPTableReusePassive(options *Options) (net.Conn, string) {
	_, localAddr, _ := utils.CheckIPPort(options.Listen)
	setReuseSecret(options)
	setPortReuseRules(localAddr, options.ReusePort)
	conn, uuid := NormalPassive(options, 0, protocol.TEMP_UUID)
	return conn, uuid
}

func setReuseSecret(options *Options) {
	firstSecret := utils.GetStringMd5(options.Secret)
	secondSecret := utils.GetStringMd5(firstSecret)
	finalSecret := firstSecret[:24] + secondSecret[:24]
	START_FORWARDING = finalSecret[16:32]
	STOP_FORWARDING = finalSecret[32:]
}

func deletePortReuseRules(localPort string, reusedPort string) error {
	var cmds []string

	cmds = append(cmds, fmt.Sprintf("iptables -t nat -D PREROUTING -p tcp --dport %s --syn -m recent --rcheck --seconds 3600 --name %s --rsource -j %s", reusedPort, strings.ToLower(CHAIN_NAME), CHAIN_NAME))
	cmds = append(cmds, fmt.Sprintf("iptables -D INPUT -p tcp -m string --string %s --algo bm -m recent --name %s --remove -j ACCEPT", STOP_FORWARDING, strings.ToLower(CHAIN_NAME)))
	cmds = append(cmds, fmt.Sprintf("iptables -D INPUT -p tcp -m string --string %s --algo bm -m recent --set --name %s --rsource -j ACCEPT", START_FORWARDING, strings.ToLower(CHAIN_NAME)))
	cmds = append(cmds, fmt.Sprintf("iptables -t nat -F %s", CHAIN_NAME))
	cmds = append(cmds, fmt.Sprintf("iptables -t nat -X %s", CHAIN_NAME))

	for _, each := range cmds {
		cmd := strings.Split(each, " ")
		err := exec.Command(cmd[0], cmd[1:]...).Run()
		if err != nil {
			log.Println("[*]Error!Use the '" + each + "' to delete rules.")
		}
	}

	fmt.Println("[*]All rules have been cleared successfully!")

	return nil
}

func setPortReuseRules(localAddr string, reusedPort string) error {
	sigs := make(chan os.Signal, 1)

	localPort := utils.GiveMePortViaAddr(localAddr)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM) //监听ctrl+c、kill命令
	go func() {
		for {
			<-sigs
			deletePortReuseRules(localPort, reusedPort)
			os.Exit(0)
		}
	}()

	var cmds []string
	cmds = append(cmds, fmt.Sprintf("iptables -t nat -N %s", CHAIN_NAME))                                                                                                                                      //新建自定义链
	cmds = append(cmds, fmt.Sprintf("iptables -t nat -A %s -p tcp -j REDIRECT --to-port %s", CHAIN_NAME, localPort))                                                                                           //将自定义链定义为转发流量至自定义监听端口
	cmds = append(cmds, fmt.Sprintf("iptables -A INPUT -p tcp -m string --string %s --algo bm -m recent --set --name %s --rsource -j ACCEPT", START_FORWARDING, strings.ToLower(CHAIN_NAME)))                  //设置当有一个报文带着特定字符串经过INPUT链时，将此报文的源地址加入一个特定列表中
	cmds = append(cmds, fmt.Sprintf("iptables -A INPUT -p tcp -m string --string %s --algo bm -m recent --name %s --remove -j ACCEPT", STOP_FORWARDING, strings.ToLower(CHAIN_NAME)))                          //设置当有一个报文带着特定字符串经过INPUT链时，将此报文的源地址从一个特定列表中移除
	cmds = append(cmds, fmt.Sprintf("iptables -t nat -A PREROUTING -p tcp --dport %s --syn -m recent --rcheck --seconds 3600 --name %s --rsource -j %s", reusedPort, strings.ToLower(CHAIN_NAME), CHAIN_NAME)) // 设置当有任意报文访问指定的复用端口时，检查特定列表，如果此报文的源地址在特定列表中且不超过3600秒，则执行自定义链

	for _, each := range cmds {
		cmd := strings.Split(each, " ")
		err := exec.Command(cmd[0], cmd[1:]...).Run() //添加规则
		if err != nil {
			return err
		}
	}

	return nil
}

func SoReusePassive(options *Options, isReconnect uint16, uuid string) (net.Conn, string) {
	listenAddr := fmt.Sprintf("%s:%s", options.ReuseHost, options.ReusePort)

	listener, err := reuseport.Listen("tcp", listenAddr)
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
		UUIDLen:     uint16(len(uuid)),
		UUID:        uuid,
		IsAdmin:     0,
		IsReconnect: isReconnect,
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

		if err := share.PassivePreAuth(conn, options.Secret); err != nil {
			log.Fatalf("[*]Error occured: %s", err.Error())
		}

		rMessage = protocol.PrepareAndDecideWhichRProtoFromUpper(conn, options.Secret, protocol.TEMP_UUID)
		fHeader, fMessage, err := protocol.DestructMessage(rMessage)

		if err != nil {
			log.Printf("[*]Fail to set connection from %s, Error: %s\n", conn.RemoteAddr().String(), err.Error())
			conn.Close()
			continue
		}

		if fHeader.MessageType == protocol.HI {
			mmess := fMessage.(*protocol.HIMess)
			if mmess.Greeting == "Shhh..." && mmess.IsAdmin == 1 {
				sMessage = protocol.PrepareAndDecideWhichSProtoToUpper(conn, options.Secret, protocol.TEMP_UUID)
				protocol.ConstructMessage(sMessage, header, hiMess, false)
				sMessage.SendMessage()
				uuid := achieveUUID(conn, options.Secret)
				log.Printf("[*]Connection from admin %s is set up successfully!\n", conn.RemoteAddr().String())
				return conn, uuid
			}
		}

		conn.Close()
		log.Println("[*]Incoming connection seems illegal!")
	}
}
