package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"
)

type ChatClient struct {
	Connection       net.Conn
	Nickname         string
	ConnectionString string
}

func NewChatClient(connectionString, nickname string) ChatClient {
	return ChatClient{
		ConnectionString: connectionString,
		Nickname:         nickname,
	}
}

func (c *ChatClient) Connect() (err error) {
	c.Connection, err = net.Dial("tcp", c.ConnectionString)
	return err
}

func (c *ChatClient) Login() (err error) {
	_, err = fmt.Fprintf(c.Connection, "%s\r\n", c.Nickname)
	return err
}

func (c *ChatClient) Close() error {
	return c.Connection.Close()
}

func (c *ChatClient) Listen() {
	for {
		message, err := bufio.NewReader(c.Connection).ReadString('\n')
		if err, ok := err.(net.Error); ok && err.Timeout() {
			log.Fatal("Timeout. Maybe something wrong with network?")
		}
		if err == io.EOF {
			log.Fatal("Server has stopped")
		}
		if err != nil {
			log.Fatal(err)
		}
		c.Connection.SetReadDeadline(time.Time{})
		fmt.Print(message)
	}
}

func (c *ChatClient) InputLoop() error {
	for {
		reader := bufio.NewReader(os.Stdin)
		text, err := reader.ReadString('\n')
		if err != nil {
			return err
		}

		_, err = fmt.Fprintf(c.Connection, "%s", text)
		if err != nil {
			return err
		}
		// на случай если отвалится сеть
		err = c.Connection.SetReadDeadline(time.Now().Add(time.Second * 5))
		if err != nil {
			return err
		}
	}
}

func main() {
	if len(os.Args) != 3 {
		fmt.Println("Wrong Argument count")
		fmt.Println("Usage: client <connectionString> <nickname>")
		os.Exit(1)
	}

	client := NewChatClient(os.Args[1], os.Args[2])

	err := client.Connect()
	if err != nil {
		log.Fatal(err.Error())
	}

	defer client.Close()

	err = client.Login()
	if err != nil {
		log.Fatal(err.Error())
	}

	go client.Listen()

	err = client.InputLoop()
	if err != nil {
		log.Fatal(err.Error())
	}
}
