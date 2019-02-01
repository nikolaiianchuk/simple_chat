package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
)

const ServerUsername = "server"

type Message struct {
	SenderName string
	Text       string
}

type ChatServer struct {
	l                 net.Listener
	UserConnections   map[string][]net.Conn
	UserConnectionMtx sync.Mutex
	OpenedConnections chan net.Conn
	DeadConnections   chan UserConnection
	Messages          chan Message
}

type UserConnection struct {
	Username   string
	Connection net.Conn
}

func NewChatServer() ChatServer {
	return ChatServer{
		UserConnections:   make(map[string][]net.Conn),
		UserConnectionMtx: sync.Mutex{},
		OpenedConnections: make(chan net.Conn, 1000),
		DeadConnections:   make(chan UserConnection, 1000),
		Messages:          make(chan Message, 1000),
	}
}

func (c *ChatServer) Listen(port int) (err error) {
	c.l, err = net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", port))
	if err != nil {
		return err
	}
	defer c.Close()

	go c.AcceptNewConnections()
	c.ServeConnections()

	return nil
}

func (c *ChatServer) Close() {
	err := c.l.Close()
	if err != nil {
		log.Println(err.Error())
	}

	for _, userConns := range c.UserConnections {
		for _, conn := range userConns {
			err = conn.Close()
			if err != nil {
				log.Println(err.Error())
			}
		}
	}
}

func (c *ChatServer) AcceptNewConnections() {
	for {
		conn, err := c.l.Accept()
		if err != nil {
			log.Println(err.Error())
			continue
		}
		c.OpenedConnections <- conn
	}
}

func (c *ChatServer) ServeConnections() {
	for {
		select {
		case conn := <-c.OpenedConnections:
			go func(conn net.Conn) {
				r := bufio.NewReader(conn)
				senderName, err := r.ReadString('\n')
				if err != nil {
					return
				}
				senderName = strings.Trim(senderName, "\r\n")

				if isNewUser := c.addUserClient(senderName, conn); isNewUser {
					c.Messages <- Message{SenderName: ServerUsername, Text: fmt.Sprintf("%s is online\r\n", senderName)}
				}

				for {
					text, err := r.ReadString('\n')
					if err != nil {
						break
					}
					c.Messages <- Message{SenderName: senderName, Text: text}
				}

				c.DeadConnections <- UserConnection{Username: senderName, Connection: conn}
			}(conn)
		case deadConn := <-c.DeadConnections:
			if deleted := c.deleteUserClient(deadConn.Username, deadConn.Connection); deleted {
				c.Messages <- Message{SenderName: ServerUsername, Text: fmt.Sprintf("%s is offline\r\n", deadConn.Username)}
			}
		case msg := <-c.Messages:
			for _, userConns := range c.UserConnections {
				for _, conn := range userConns {
					_, err := conn.Write([]byte(fmt.Sprintf("%s: %s", msg.SenderName, msg.Text)))
					if err != nil {
						log.Println(err.Error())
					}
				}
			}
			log.Print(fmt.Sprintf("%s: %s", msg.SenderName, msg.Text))
		}
	}
}

func (c *ChatServer) addUserClient(username string, conn net.Conn) (isNew bool) {
	c.UserConnectionMtx.Lock()
	defer c.UserConnectionMtx.Unlock()

	if _, ok := c.UserConnections[username]; !ok {
		c.UserConnections[username] = []net.Conn{conn}
		return true
	}
	c.UserConnections[username] = append(c.UserConnections[username], conn)
	return false
}

func (c *ChatServer) deleteUserClient(username string, conn net.Conn) (isDeleted bool) {
	c.UserConnectionMtx.Lock()
	defer c.UserConnectionMtx.Unlock()

	var id int
	for i, c := range c.UserConnections[username] {
		if c == conn {
			id = i
			break
		}
	}

	c.UserConnections[username] = append(c.UserConnections[username][:id], c.UserConnections[username][id+1:]...)

	if len(c.UserConnections[username]) == 0 {
		delete(c.UserConnections, username)
		return true
	}
	return false
}

func main() {
	cs := NewChatServer()
	err := cs.Listen(50000)
	if err != nil {
		log.Fatal(err)
	}
}
