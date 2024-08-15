package netwrk

import (
	"fmt"
	"io"
	"log"
	"net"

	"google.golang.org/protobuf/proto"
)

func handleLobbyConnection(conn net.Conn) {
	defer conn.Close()

	messageBytes := make([]byte, 4096)

	recvMessageChan := make(chan *LobbyMessage)
	go func() {
		for {
			fmt.Println("READING!")
			n, err := conn.Read(messageBytes)
			if err == io.EOF {
				return
			}
			fmt.Println("READ something!")
			if err != nil {
				log.Printf("Error reading message %v", err)
				return
			}

			message := LobbyMessage{}

			err = proto.Unmarshal(messageBytes[:n], &message)
			if err != nil {
				log.Println("Invalid message received from client")
			}
			recvMessageChan <- &message
		}
	}()

	for {

		select {
		case msg := <-recvMessageChan:
			if isDone, err := handleClientLobbyMessage(conn, msg); err != nil || isDone {
				log.Println(err)
				return
			}
			fmt.Println("Handled message")
		}
	}
}

// Returns a bool of whether the player has disconnected from the lobby and an error
func handleClientLobbyMessage(playerConnection net.Conn, message *LobbyMessage) (bool, error) {
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

		log.Println("Broadcasting new player", message.Content)

		broadcastToLobby(&LobbyMessage{PlayerId: "", Type: "connect", Content: playerID})

		return false, SendMessageToClient(playerConnection, &LobbyMessage{PlayerId: playerID, Type: "name", Content: playerID})
	case "invite":
		log.Println("Got invite for player:", message.Content)
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
	case "chat":
		broadcastToLobby(&LobbyMessage{PlayerId: message.PlayerId, Type: "text", Content: message.Content})
		return false, nil
	case "decline_game":
		inviter := clientPool.clients[message.Content]
		SendMessageToClient(inviter.conn, &LobbyMessage{Type: "decline_game", Content: message.PlayerId})
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
	fmt.Println("Sent message to client")
	return nil
}

func broadcastToLobby(message *LobbyMessage) {
	for _, player := range clientPool.clients {
		err := SendMessageToClient(player.conn, message)
		if err != nil {
			log.Println("Error broadcasting to clients...", err)
		}
	}
}
