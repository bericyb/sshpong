package main

import (
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"
	"os"
	"sshpong/internal/client"
	"sshpong/internal/lobby"
	"strings"
)

var username string

func main() {
	slog.SetLogLoggerLevel(slog.LevelDebug)
	slog.Debug("Debug logs active...")

	fmt.Println("Welcome to sshpong!")
	fmt.Println("Please enter your username")

	egress := make(chan lobby.LobbyMessage)
	ingress := make(chan lobby.LobbyMessage)
	interrupter := make(chan client.InterrupterMessage, 100)
	exit := make(chan string)

	buf := make([]byte, 1024)
	n, err := os.Stdin.Read(buf)
	if err != nil {
		log.Panic("Bro your input is no good...")
	}
	username = string(buf[:n-1])

	fmt.Println("username is...", username)
	conn, err := ConnectToLobby(username)
	if err != nil {
		log.Panic(err)
	}

	// User input handler
	go func(egress chan lobby.LobbyMessage) {
		buf := make([]byte, 1024)
		for {
			n, err := os.Stdin.Read(buf)
			if err != nil {
				log.Panic("Bro your input wack as fuck")
			}

			input := string(buf[:n-1])
			args := strings.Fields(input)

			userMessage := lobby.LobbyMessage{}

			select {
			case msg := <-interrupter:
				userMessage, err := client.HandleInterruptInput(msg, args, username)
				if err != nil {
					userMessage, err = client.HandleUserInput(args, username)
					if err == io.EOF {
						exit <- ""
					}
					if err != nil {
						fmt.Println(err)
						continue
					}
				}
				egress <- userMessage
				if userMessage.MessageType == "accept" || userMessage.MessageType == "disconect" {
					slog.Debug("Closing input handler with accept or disconnect message", slog.Any("message content", userMessage.Message))
					return
				}
				if userMessage.MessageType == "start_game" {
					slog.Debug("closing input handler with start_game message and sending exit signal")

					// TODO: This is a wierd one...
					sg, ok := userMessage.Message.(lobby.StartGame)
					if !ok {
						slog.Debug("Start game interrupt message was improperly formatted... Could be indicative of an error in the HandleinterruptInput method")
						continue
					}
					exit <- sg.GameID
					return
				}

			default:
				userMessage, err = client.HandleUserInput(args, username)
				if err == io.EOF {
					exit <- ""
				}
				if err != nil {
					fmt.Println(err)
					continue
				}
				egress <- userMessage

			}

		}
	}(egress)

	// Ingress Handler
	go func(oc chan lobby.LobbyMessage) {
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
	go func(userMessages chan lobby.LobbyMessage) {
		for {
			msg := <-userMessages
			bytes, err := lobby.Marshal(msg)
			if err != nil {
				log.Panic("Malformed proto message", err)
			}
			_, err = conn.Write(bytes)
			if err == io.EOF {
				log.Panic("Server disconnected sorry...")
			} else if err != nil {
				log.Panic("Error reading from server connection...")
			}
			if msg.MessageType == "start_game" || msg.MessageType == "disconnect" {
				slog.Debug("closing network writer ")
				return
			}
		}
	}(egress)

	// Network reader
	go func(serverMessages chan lobby.LobbyMessage) {
		buf := make([]byte, 1024)
		for {
			n, err := conn.Read(buf)
			if err == io.EOF {
				fmt.Println("disconnected from lobby")
			} else if err != nil {
				log.Panic("Error reading from server connection...", err)
			}

			message, err := lobby.Unmarshal(buf[:n])
			if err != nil {
				log.Panic("Error reading message from server", err)
			}
			serverMessages <- message
		}
	}(ingress)

	fmt.Println("Waiting for an exit message")
	isStartGame := <-exit
	if isStartGame != "" {
		fmt.Println("Connecting to game", isStartGame)
		gameConn, err := ConnectToGame(username, isStartGame)

		if err != nil {
			log.Panic("Failed to connect to game server...", err)
		}

		client.Game(gameConn)

	} else {
		return
	}
}

func ConnectToLobby(username string) (net.Conn, error) {
	conn, err := net.Dial("tcp", "127.0.0.1:12345")
	if err != nil {
		return nil, fmt.Errorf("Sorry, failed to connect to server...")
	}

	loginMsg, err := lobby.Marshal(lobby.LobbyMessage{MessageType: "name", Message: lobby.Name{Name: username}})
	if err != nil {
		return nil, fmt.Errorf("Sorry bro but your username is wack AF...")
	}

	_, err = conn.Write(loginMsg)
	if err != nil {
		return nil, fmt.Errorf("Sorry, could not communicate with server...")
	}

	return conn, nil
}

func ConnectToGame(username, gameID string) (net.Conn, error) {
	conn, err := net.Dial("tcp", "127.0.0.1:42069")
	if err != nil {
		return nil, err

	}

	_, err = conn.Write([]byte(fmt.Sprintf("%s:%s", gameID, username)))
	if err != nil {
		return nil, err
	}

	return conn, nil
}
