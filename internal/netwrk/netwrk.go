package netwrk

import (
	"fmt"
	"log"
	"net"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
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
	client1 Client
	client2 Client
}

type GameConnections struct {
	games map[string]GameClients
}

var clientPool *ClientPool

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
}

func handleGameConnection(conn net.Conn) {
	defer conn.Close()

	messageBytes := make([]byte, 126)

	for {
		n, err := conn.Read(messageBytes)
		if err != nil {
			log.Printf("Error reading message %v", err)
			return
		}

		if isDone, err := handleGameMessage(conn, messageBytes[:n]); err != nil {
			return
		}
	}
}

func handleGameMessage(conn net.Conn, message GameMessage) error {

	return nil
}

func handleLobbyConnection(conn net.Conn) {
	defer conn.Close()

	messageBytes := make([]byte, 4096)

	for {
		n, err := conn.Read(messageBytes)
		if err != nil {
			log.Printf("Error reading message %v", err)
			return
		}

		if isDone, err := handleLobbyMessage(conn, messageBytes[:n]); err != nil || isDone {
			return
		}
	}
}

// Returns a bool of whether the player has disconnected from the lobby and an error
func handleLobbyMessage(playerConnection net.Conn, bytes []byte) (bool, error) {

	message := LobbyMessage{}

	err := proto.Unmarshal(bytes, &message)
	if err != nil {
		return false, fmt.Errorf("Invalid message received from client")
	}

	switch message.Type {
	case "name":
		clientPool.clients[uuid.New().String()] = Client{
			name:  message.Content,
			conn:  playerConnection,
			ready: false,
		}
		break
	case "invite_player":
		invitee, ok := clientPool.clients[message.Content]
		if !ok {
			SendMessageToClient(playerConnection, &LobbyMessage{Type: "text", Content: "Sorry that player is not available..."})
			return false, nil
		}
		SendMessageToClient(invitee.conn, &LobbyMessage{Type: "invite", Content: playerID})
		return false, nil
	case "cancel_invite":

	case "accept_game":
		AcceptGame(message.Content)
		return true, nil
	case "decline_game":
		DeclineGame()
		return false, nil
	case "quit":
		DeletePlayer(message.Content)
		return true, nil
	case "ping":
		PongPlayer(message.Content)
		return false, nil
	default:
		PongPlayer(message.Content)
		return false, nil
	}
	return false, nil

}

func SendMessageToClient(connection net.Conn, message *LobbyMessage) error {
	bytes, err := proto.Marshal(message)
	if err != nil {
		return fmt.Errorf("Error marshalling message. Your protobuf is wack yo.")
	}
	_, err = connection.Write(bytes)
	if err != nil {
		return fmt.Errorf("Error writing to client connection")
	}
	return nil
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

func SendGameUpdateToLobbyClients(gameID string, message *LobbyMessage) error {
	clients, ok := gameConnections.games[gameID]
	if !ok {
		return fmt.Errorf("Could not find game clients record")
	}

	bytes, err := proto.Marshal(message)
	if err != nil {
		return fmt.Errorf("message could not be marshalled")
	}

	_, err = clients.client1.conn.Write(bytes)
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
