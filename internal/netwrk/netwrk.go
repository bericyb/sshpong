package netwrk

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/google/uuid"
)

type Client struct {
	name string
	conn net.Conn
}

type ClientPool struct {
	clients map[string]Client
}

type GameClients struct {
	client1 Client
	client2 Client
}

type GameConnections struct {
	games map[string]GameClients
}

var clientPool *ClientPool
var gameConnections *GameConnections

// Starts listening on port 12345 for TCP connections
// Also creates client pool and game connection singletons
func Listen() {
	clientPool = &ClientPool{
		clients: map[string]Client{},
	}

	gameConnections = &GameConnections{
		games: map[string]GameClients{},
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

			name := make([]byte, 1024)

			n, err := conn.Read(name)
			if err != nil {
				log.Println(fmt.Sprintf("Failed to read from connection: %s", conn.LocalAddr()))
			} else {
				clientPool.clients[uuid.New().String()] = Client{
					name: string(name[:n]),
					conn: conn,
				}
			}
		}
	}()

}

func handleLobbyConnection(connID string, conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)

	for {
		message, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("Error reading message %v", err)
			delete(clientPool.clients, connID)
			return
		}

		handleLobbyMessage(message, conn)
	}
}

func handleLobbyMessage(message string, conn net.Conn) {

	key, data := decodeMessage(message)
	switch message {
	case "start":
		break
	case "waiting":
		break
	default:
		clientPool.clients[message]
	}
}

func decodeLobbyMessage(message string) (string, any) {
	switch message {
	case "start":
		return "start", nil
	case "waiting":
		return "waiting", nil
	default:
	break
	}

	strings

	}
}

func GetPool() (map[string]Client, bool) {
	if clientPool.clients != nil {
		return clientPool.clients, true
	}
	return clientPool.clients, false
}

func CreateGame(clientID1, clientID2 string) (string, error) {
	client1, ok := clientPool.clients[clientID1]
	if !ok {
		return "", fmt.Errorf("Client 1 was not found in client pool :(")
	}
	if err := client1.conn.SetWriteDeadline(time.Time{}); err != nil {
		return "", fmt.Errorf("Client 1 was not responsive")
	}

	client2, ok := clientPool.clients[clientID2]
	if !ok {
		return "", fmt.Errorf("Client 2 was not found in client pool :(")
	}
	if err := client2.conn.SetWriteDeadline(time.Time{}); err != nil {
		return "", fmt.Errorf("Client 2 was not responsive")
	}

	gameID := uuid.New().String()
	gameConnections.games[gameID] = GameClients{
		client1: client1,
		client2: client2,
	}

	return gameID, nil
}

func SendGameUpdateToClients(gameID string, bytes []byte) error {
	clients, ok := gameConnections.games[gameID]
	if !ok {
		return fmt.Errorf("Could not find game clients record")
	}

	_, err := clients.client1.conn.Write(bytes)
	if err != nil {
		return fmt.Errorf("Could not write to client1 connection")
	}
	_, err = clients.client1.conn.Write(bytes)
	if err != nil {
		return fmt.Errorf("Could not write to client2 connection")
	}

	return nil
}

func PingAndCleanLobbyClients() {
	ping := []byte("ping")

	deadClients := []string{}
	for id, client := range clientPool.clients {
		client.conn.SetWriteDeadline(time.Now().Add(time.Second))
		_, err := client.conn.Write(ping)
		if err != nil {
			log.Println("Could not write to client, deleting connection:", id)
			deadClients = append(deadClients, id)
		}
	}

	for _, id := range deadClients {
		delete(clientPool.clients, id)
	}
}
