package main

import (
	"fmt"
	"log"
	"log/slog"
	"net"
	"os"
	"sshpong/internal/config"
	"sshpong/internal/lobby"
	"sshpong/internal/pong"
	"strings"
	"sync"
)

var exit chan bool
var games sync.Map

func main() {
	if len(os.Args) == 1 {
		config.LoadConfig("")
	} else {
		config.LoadConfig(os.Args[1])
	}

	slog.SetLogLoggerLevel(slog.Level(config.Config.LogLevel))
	fmt.Println("Starting sshpong lobby...")
	go LobbyListen()
	fmt.Println("Lobby started")

	fmt.Println("Starting game listener...")
	go GamesListen()
	fmt.Println("Game listener started")

	_ = <-exit
}

// Starts listening on port 12345 for TCP connections
// Also creates client pool and game connection singletons
func LobbyListen() {
	listener, err := net.Listen("tcp", "127.0.0.1:12345")
	if err != nil {
		slog.Error("Error setting up listener for lobby. Exiting...", err)
	}

	defer listener.Close()

	l := lobby.CreateLobby()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println(err)
			continue
		}

		go func() {
			client, msgOut, err := l.InitialConnectionHandler(conn)
			if err != nil {
				fmt.Println("what?")
				conn.Write(msgOut)
			} else {
				fmt.Println("new client", client, err)
				_, err = conn.Write(msgOut)
				if err != nil {
					slog.Debug("error writing to new player... disconnecting")
					msg, err := lobby.Marshal(lobby.DisconnectData{
						From: client.Username,
					}, lobby.Disconnect)
					if err != nil {
						slog.Error("error marshalling disconnect message on player connect")
					}
					l.BroadcastToLobby(msg)
				}

				go l.HandleLobbyConnection(client)
			}
		}()
	}
}

func GamesListen() {

	type GameClients struct {
		Client1 lobby.Client
		Client2 lobby.Client
	}

	gameListener, err := net.Listen("tcp", "127.0.0.1:42069")
	if err != nil {
		log.Fatal(err)
	}
	defer gameListener.Close()

	for {
		conn, err := gameListener.Accept()
		if err != nil {
			log.Println(err)
			continue
		}

		slog.Debug("Received game connection")

		go func(conn net.Conn) {
			messageBytes := make([]byte, 126)

			n, err := conn.Read(messageBytes)
			if err != nil {
				log.Printf("Error reading game ID on connection %s", err)
			}

			gInfo := strings.SplitAfter(string(messageBytes[:n]), ":")
			if err != nil {
				log.Printf("Game id was not a string? %s", err)
			}

			slog.Debug("Game request data", slog.Any("game info", gInfo))

			game, ok := games.Load(gInfo[0])
			if !ok {
				games.Store(gInfo[0], GameClients{Client1: lobby.Client{
					Username: gInfo[1],
					Conn:     conn,
				}, Client2: lobby.Client{}})
			} else {
				gameclients, _ := game.(GameClients)
				client2 := lobby.Client{
					Username: gInfo[1],
					Conn:     conn,
				}

				games.Store(gInfo[0], GameClients{
					Client1: gameclients.Client1,
					Client2: client2})

				go pong.StartGame(gameclients.Client1.Conn, client2.Conn, gameclients.Client1.Username, client2.Username)
			}
		}(conn)
	}
}
