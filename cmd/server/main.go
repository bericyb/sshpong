package main

import (
	"fmt"
	"log"
	"log/slog"
	"net"
	"sshpong/internal/lobby"
	"sync"
)

var exit chan bool
var games sync.Map

func main() {
	slog.SetLogLoggerLevel(slog.LevelDebug)
	fmt.Println("Starting sshpong lobby...")
	go LobbyListen()
	fmt.Println("Lobby started")

	// fmt.Println("Starting game listener...")
	// go GamesListen()
	// fmt.Println("Game listener started")

	_ = <-exit
}

// Starts listening on port 12345 for TCP connections
// Also creates client pool and game connection singletons
func LobbyListen() {

	listener, err := net.Listen("tcp", "127.0.0.1:12345")
	if err != nil {
		slog.Error("Error setting up listener for lobby. Exiting...")
	}

	defer listener.Close()

	l := lobby.CreateLobby()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		go l.HandleLobbyConnection(conn)
	}
}

// func GamesListen() {
//
// 	slog.SetLogLoggerLevel(slog.LevelDebug)
// 	slog.Debug("Debug level logs are active")
//
// 	gameListener, err := net.Listen("tcp", "127.0.0.1:42069")
// 	if err != nil {
// 		log.Fatal(err)
// 	}
//
// 	for {
// 		defer gameListener.Close()
// 		conn, err := gameListener.Accept()
// 		if err != nil {
// 			log.Println(err)
// 			continue
// 		}
//
// 		slog.Debug("Received game connection")
//
// 		go func(conn net.Conn) {
// 			messageBytes := make([]byte, 126)
//
// 			n, err := conn.Read(messageBytes)
// 			if err != nil {
// 				log.Printf("Error reading game ID on connection %s", err)
// 			}
//
// 			gInfo := strings.SplitAfter(string(messageBytes[:n]), ":")
// 			if err != nil {
// 				log.Printf("Game id was not a string? %s", err)
// 			}
//
// 			slog.Debug("Game request data", slog.Any("game info", gInfo))
//
// 			game, ok := games.Load(gInfo[0])
// 			if !ok {
// 				games.Store(gInfo[0], GameClients{Client1: Client{
// 					Username: gInfo[1],
// 					Conn:     conn,
// 				}, Client2: Client{}})
// 			} else {
// 				gameclients, _ := game.(GameClients)
// 				client2 := Client{
// 					Username: gInfo[1],
// 					Conn:     conn,
// 				}
//
// 				games.Store(gInfo[0], GameClients{
// 					Client1: gameclients.Client1,
// 					Client2: client2})
//
// 				go pong.StartGame(gameclients.Client1.Conn, client2.Conn, gameclients.Client1.Username, client2.Username)
// 			}
// 		}(conn)
// 	}
// }
