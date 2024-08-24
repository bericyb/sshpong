package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"sshpong/internal/client"
	"sshpong/internal/netwrk"
	"strings"

	"google.golang.org/protobuf/proto"
)

var exit chan bool

func main() {
	fmt.Println("Welcome to sshpong!")
	fmt.Println("Please enter your username")

	egress := make(chan *netwrk.LobbyMessage)
	ingress := make(chan *netwrk.LobbyMessage)
	interrupter := make(chan client.InterrupterMessage, 100)

	buf := make([]byte, 1024)
	n, err := os.Stdin.Read(buf)
	if err != nil {
		log.Panic("Bro your input is no good...")
	}
	username := string(buf[:n-1])

	conn, err := netwrk.ConnectToLobby(username)
	if err != nil {
		log.Panic(err)
	}

	// User input handler
	go func(egress chan *netwrk.LobbyMessage) {
		buf := make([]byte, 1024)
		for {

			n, err := os.Stdin.Read(buf)
			if err != nil {
				log.Panic("Bro your input wack as fuck")
			}

			input := string(buf[:n-1])
			args := strings.Fields(input)

			userMessage := &netwrk.LobbyMessage{}

			select {
			case msg := <-interrupter:
				userMessage, err := client.HandleInterruptInput(msg, args)
				if err != nil {
					userMessage, err = client.HandleUserInput(args)
					if err == io.EOF {
						exit <- true
					}
					if err != nil {
						fmt.Println(err)
						continue
					}
				}
				userMessage.PlayerId = username
				egress <- userMessage

			default:
				userMessage, err = client.HandleUserInput(args)
				if err == io.EOF {
					exit <- true
				}
				if err != nil {
					fmt.Println(err)
					continue
				}
				userMessage.PlayerId = username
				egress <- userMessage

			}
		}
	}(egress)

	// Ingress Handler
	go func(oc chan *netwrk.LobbyMessage) {
		for {
			msg := <-ingress

			interrupterMsg, err := client.HandleServerMessage(msg)
			if err != nil {
				log.Panic("Error handling server message disconnecting...")
			}
			if interrupterMsg.InterruptType != "" {
				interrupter <- interrupterMsg
			}
		}

	}(ingress)

	// Network writer
	go func(userMessages chan *netwrk.LobbyMessage) {
		for {
			msg := <-userMessages
			bytes, err := proto.Marshal(msg)
			if err != nil {
				log.Panic("Malformed proto message", err)
			}
			_, err = conn.Write(bytes)
			if err == io.EOF {
				log.Panic("Server disconnected sorry...")
			} else if err != nil {
				log.Panic("Error reading from server connection...")
			}
		}
	}(egress)

	// Network reader
	go func(serverMessages chan *netwrk.LobbyMessage) {
		buf := make([]byte, 1024)
		for {
			n, err := conn.Read(buf)
			if err == io.EOF {
				log.Panic("Server disconnected sorry...")
			} else if err != nil {
				log.Panic("Error reading from server connection...", err)
			}

			message := &netwrk.LobbyMessage{}

			err = proto.Unmarshal(buf[:n], message)
			if err != nil {
				log.Panic("Error reading message from server")
			}

			serverMessages <- message

		}
	}(ingress)

	_ = <-exit
}
