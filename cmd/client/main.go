package main

import (
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"
	"os"
	"sshpong/internal/client"
	"sshpong/internal/config"
	"sshpong/internal/lobby"
	"strings"
)

var username string

func main() {
	if len(os.Args) == 1 {
		config.LoadConfig("")
	} else {
		config.LoadConfig(os.Args[1])
	}

	slog.SetLogLoggerLevel(slog.Level(config.Config.LogLevel))

	fmt.Println("Welcome to sshpong!")

	egress := make(chan []byte)
	ingress := make(chan []byte)
	interrupter := make(chan client.InterrupterMessage, 100)
	exit := make(chan string)

	var usernameOk = false

	// In the future make a DB call as well?
	isUsernameOk := func(un string) bool {
		if strings.Contains(un, ":") || len(strings.Split(un, " ")) > 1 || len(un) < 1 {
			fmt.Println(client.Red, "Sorry, please pick a username that has no special characters or spaces.", client.Normal)
			return false
		}
		return true
	}

	for !usernameOk {
		fmt.Println("Please enter your username")
		buf := make([]byte, 1024)
		n, err := os.Stdin.Read(buf)
		if err != nil {
			log.Panic("Bro your input is no good...")
		}
		username = string(buf[:n-1])
		usernameOk = isUsernameOk(username)
	}

	conn, err := ConnectToLobby(username)
	if err != nil {
		log.Panic(err)
	}

	// User input handler
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := os.Stdin.Read(buf)
			if err != nil {
				log.Panic("Bro your input wack as fuck")
			}

			input := string(buf[:n-1])
			args := strings.Fields(input)

			userMessage := []byte{}

			select {
			case msg := <-interrupter:
				if msg.InterruptType == "start_game" {
					slog.Debug("closing input handler with start_game message and sending exit signal")
					exit <- msg.Content
					return
				}
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
	}()

	// Ingress Handler
	go func() {
		for {
			msg := <-ingress

			interrupterMsg, err := client.HandleServerMessage(msg)
			if err != nil {
				log.Panic("Error handling server message disconnecting...")
			}
			if interrupterMsg.InterruptType == "start_game" {
				exit <- interrupterMsg.Content
			}
			if interrupterMsg.InterruptType != "" {
				interrupter <- interrupterMsg
			}
		}

	}()

	// Network writer
	go func() {
		for {
			msg := <-egress
			slog.Debug("writing egress message to server", "message", msg)

			_, err = conn.Write(msg)
			if err == io.EOF {
				log.Panic("Server disconnected, sorry...")
			}
		}
	}()

	// Network reader
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := conn.Read(buf)
			if err == io.EOF {
				log.Panic("disconnected from lobby")
			}
			ingress <- buf[:n]
		}
	}()

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
	slog.Debug("connecting to server...")
	conn, err := net.Dial("tcp", "127.0.0.1:12345")
	if err != nil {
		return nil, fmt.Errorf("Sorry, failed to connect to server...")
	}

	loginMsg, err := lobby.Marshal(lobby.NameData{Name: username}, lobby.Name)
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
