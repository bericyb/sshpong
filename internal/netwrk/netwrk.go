package netwrk

import (
	"log"
	"net"
)

type Client struct {
	name  string
	conn  net.Conn
	ready bool
}

type ClientPool struct {
	clients map[string]Client
}

type GameClients struct {
	client1 chan GameMessage
	client2 chan GameMessage
}

type GameChans struct {
	games map[string]GameClients
}

var clientPool *ClientPool
var gameChans *GameChans

// Starts listening on port 12345 for TCP connections
// Also creates client pool and game connection singletons
func Listen() {
	clientPool = &ClientPool{
		clients: map[string]Client{},
	}

	listener, err := net.Listen("tcp", ":12345")
	if err != nil {
		log.Fatal(err)
	}

	defer listener.Close()

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				log.Println(err)
				continue
			}
			go handleLobbyConnection(conn)
		}
	}()

	gameChans = &GameChans{
		games: map[string]GameClients{},
	}

	gameListener, err := net.Listen("tcp", ":42069")
	if err != nil {
		log.Fatal(err)
	}

	defer gameListener.Close()
	go func() {
		for {
			conn, err := gameListener.Accept()
			if err != nil {
				log.Println(err)
				continue
			}
			handleGameConnection(conn)
		}
	}()
}
