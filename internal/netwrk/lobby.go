package netwrk

import (
	"fmt"
	"log"
	"net"
	"time"

	"google.golang.org/protobuf/proto"
)

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
		_, ok := clientPool.clients[message.Content]
		if ok {
			SendMessageToClient(playerConnection, &LobbyMessage{Type: "error", Content: "Sorry, that name is already taken"})
			return false, nil
		}
		playerID := message.Content
		clientPool.clients[playerID] = Client{
			name:  playerID,
			conn:  playerConnection,
			ready: false,
		}
		for _, player := range clientPool.clients {
			err := SendMessageToClient(playerConnection, &LobbyMessage{PlayerId: player.name, Type: "connect", Content: player.name})
			if err != nil {
				log.Println("There was an error sending the list of lobby players to client", message.Content)
			}
		}

		broadcastToLobby(&LobbyMessage{PlayerId: "", Type: "connect", Content: playerID})

		return false, SendMessageToClient(playerConnection, &LobbyMessage{PlayerId: playerID, Type: "name", Content: playerID})
	case "invite_player":
		invitee, ok := clientPool.clients[message.Content]
		if !ok {
			SendMessageToClient(playerConnection, &LobbyMessage{Type: "text", Content: "Sorry, that player is not available..."})
			return false, nil
		}
		SendMessageToClient(invitee.conn, &LobbyMessage{Type: "invite", Content: message.PlayerId})
		return false, nil
	case "accept_game":
		player := clientPool.clients[message.Content]

		if err := SendMessageToClient(player.conn, &LobbyMessage{Type: "accept", Content: ""}); err != nil {
			SendMessageToClient(playerConnection, &LobbyMessage{Type: "error", Content: "Sorry that game is no longer available..."})
			return false, nil
		}

		return true, nil
	case "decline_game":
		inviter := clientPool.clients[message.Content]
		SendMessageToClient(inviter.conn, &LobbyMessage{Type: "decline_game", Content: ""})
		return false, nil
	case "quit":
		delete(clientPool.clients, message.PlayerId)
		broadcastToLobby(&LobbyMessage{Type: "disconnect", Content: message.PlayerId})
		return true, nil
	case "ping":
		SendMessageToClient(playerConnection, &LobbyMessage{Type: "pong", Content: "pong"})
		return false, nil
	default:
		SendMessageToClient(playerConnection, &LobbyMessage{Type: "pong", Content: "pong"})
		return false, nil
	}
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

func broadcastToLobby(message *LobbyMessage) {
	for _, player := range clientPool.clients {
		SendMessageToClient(player.conn, message)
	}
}
