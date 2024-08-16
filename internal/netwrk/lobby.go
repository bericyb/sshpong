package netwrk

import (
	"io"
	"log"
	"net"

	"google.golang.org/protobuf/proto"
)

func handleLobbyConnection(conn net.Conn) {
	defer conn.Close()

	messageBytes := make([]byte, 4096)

	ingress := make(chan *LobbyMessage)
	egress := make(chan *LobbyMessage)

	// Network Reader
	go func() {
		for {
			n, err := conn.Read(messageBytes)
			if err == io.EOF {
				return
			}
			if err != nil {
				log.Printf("Error reading message %v", err)
				return
			}

			message := LobbyMessage{}

			err = proto.Unmarshal(messageBytes[:n], &message)
			if err != nil {
				log.Println("Invalid message received from client", err)
			}
			ingress <- &message
		}
	}()

	// Network Writer
	go func() {
		for {
			msg := <-egress
			bytes, err := proto.Marshal(msg)
			if err != nil {
				log.Println("Error marshalling message to send to user...", err)
			}
			_, err = conn.Write(bytes)
			if err == io.EOF {
				log.Println("User has disconnected", err)
				ingress <- &LobbyMessage{Type: "disconnect"}
			}
			if err != nil {
				log.Println("Error writing to user...", err)
			}
		}
	}()

	// Client message handler
	go func() {
		for {
			msg := <-ingress
			serverMsg, err := handleClientLobbyMessage(msg)
			if err != nil {
				log.Println("Error handling client lobby message...", err)
			}
			if serverMsg != nil {
				egress <- serverMsg
			}
		}
	}()
}

// Returns a bool of whether the player has disconnected from the lobby and an error
func handleClientLobbyMessage(message *LobbyMessage) (*LobbyMessage, error) {
	switch message.Type {
	case "name":
		_, ok := lobbyMembers.Load(message.Content)
		if ok {
			return &LobbyMessage{Type: "name_error", Content: "Sorry, that name is already taken, please try a different name"}, nil
		}
		username := message.Content

		// Send all client messages
		lobbyMembers.Range(func(lobbyUsername string, client Client) bool {
			externalMessageChan <- ExternalMessage{Target: username, Message: &LobbyMessage{Type: "connect", Content: lobbyUsername}}
			return true
		})

		log.Println("Broadcasting new player", message.Content)

		broadcastToLobby(&LobbyMessage{PlayerId: "", Type: "connect", Content: username})

		return &LobbyMessage{PlayerId: username, Type: "name", Content: username}, nil
	case "invite":
		log.Println("Got invite for player:", message.Content)
		invitee, ok := lobbyMembers[message.Content]
		if !ok {
			return &LobbyMessage{Type: "text", Content: "Sorry, that player is not available..."}, nil
		}
		return &LobbyMessage{Type: "invite", Content: message.PlayerId}, nil
	case "accept_game":
		player := lobbyMembers[message.Content]

		return &LobbyMessage{Type: "accept", Content: ""}, nil

	case "chat":
		broadcastToLobby(&LobbyMessage{PlayerId: message.PlayerId, Type: "text", Content: message.Content})
		return nil, nil
	case "decline_game":
		inviter := lobbyMembers[message.Content]
		return &LobbyMessage{Type: "decline_game", Content: message.PlayerId}, nil
	case "quit":
		delete(lobbyMembers, message.PlayerId)
		broadcastToLobby(&LobbyMessage{Type: "disconnect", Content: message.PlayerId})
		return nil, nil
	case "ping":
		return &LobbyMessage{Type: "pong", Content: "pong"}, nil
	default:
		return &LobbyMessage{Type: "pong", Content: "pong"}, nil
	}
}

func broadcastToLobby(message *LobbyMessage) {
	for _, player := range lobbyMembers {
		bytes, err := proto.Marshal(message)
		if err != nil {
			log.Println("Error marshalling broadcast message", err)
		}
		_, err = player.Conn.Write(bytes)
		if err != nil {
			log.Println("Error broadcasting to clients...", err)
		}
	}
}
