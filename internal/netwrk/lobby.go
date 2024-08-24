package netwrk

import (
	"io"
	"log"
	"net"
	"strings"

	"google.golang.org/protobuf/proto"
)

func handleLobbyConnection(conn net.Conn) {
	messageBytes := make([]byte, 4096)

	ingress := make(chan *LobbyMessage)
	egress := make(chan *LobbyMessage)

	// Network Reader
	go func() {
		for {
			n, err := conn.Read(messageBytes)
			if err == io.EOF {
				conn.Close()
				return
			}
			if err != nil {
				conn.Close()
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
				conn.Close()
				log.Println("User has disconnected", err)
				ingress <- &LobbyMessage{
					Type:    "disconnect",
					Content: msg.PlayerId,
				}
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
			serverMsg, err := handleClientLobbyMessage(msg, conn)
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
func handleClientLobbyMessage(message *LobbyMessage, conn net.Conn) (*LobbyMessage, error) {
	switch message.Type {

	// Handle an name/login message from a player
	// Store the new player in the lobbyMembers
	// Send a connection message for each of the lobbyMembers to the new player
	// Send a connection message to all members in the lobby
	case "name":
		_, ok := lobbyMembers.Load(message.Content)
		if ok {
			return &LobbyMessage{Type: "name_error", Content: "Sorry, that name is already taken, please try a different name"}, nil
		}
		username := message.Content

		lobbyMembers.Store(username, Client{Username: username, Conn: conn})

		// Build current lobby list
		var lobby []string
		lobbyMembers.Range(func(lobbyUsername any, client any) bool {
			usernameString, _ := lobbyUsername.(string)
			lobby = append(lobby, usernameString)
			return true
		})

		broadcastToLobby(&LobbyMessage{PlayerId: "", Type: "connect", Content: username})

		return &LobbyMessage{PlayerId: username, Type: "name", Content: strings.Join(lobby, "\n")}, nil

	// Handle an invite message by sending a message to the target player
	// Send an invite message to the invitee: message.Content
	// Send an ack message to the inviter: message.PlayerId
	case "invite":
		externalMessageChan <- ExternalMessage{
			Target:  message.Content,
			Message: message,
		}

		return &LobbyMessage{Type: "pending_invite", Content: message.Content}, nil

	// Handle a accept message from a player that was invited
	// Send a game_start message back to the player: message.PlayerId
	// Send an accepted message back to the inviter: message.Content
	case "accept":
		externalMessageChan <- ExternalMessage{
			Target:  message.Content,
			Message: &LobbyMessage{Type: "game_start", Content: ""},
		}

		return &LobbyMessage{PlayerId: message.PlayerId, Type: "accepted", Content: ""}, nil

	// Handle a chat message from a player with PlayerId
	case "chat":
		broadcastToLobby(&LobbyMessage{PlayerId: message.PlayerId, Type: "text", Content: message.Content})
		return nil, nil

	// Handle a decline_game message from a player that was invited
	// Send an ack message back to the invitee: message.PlayerId
	// Send an ack message to the inviter: message.Content
	case "decline_game":
		externalMessageChan <- ExternalMessage{
			Target:  message.Content,
			Message: &LobbyMessage{Type: "decline", Content: ""},
		}

		return &LobbyMessage{Type: "decline_game", Content: message.PlayerId}, nil

	// Handle a quit message from a player that was connected
	// broadcast the player quit to the lobby
	case "quit":
		lobbyMembers.Delete(message.PlayerId)
		broadcastToLobby(&LobbyMessage{Type: "disconnect", Content: message.PlayerId})
		return nil, nil

	// Ping and pong
	case "ping":
		return &LobbyMessage{Type: "pong", Content: "pong"}, nil

	// Ping and pong
	default:
		return &LobbyMessage{Type: "pong", Content: "pong"}, nil
	}
}

func broadcastToLobby(message *LobbyMessage) {
	var disconnectedUsers []string
	lobbyMembers.Range(func(playerId, player interface{}) bool {
		bytes, err := proto.Marshal(message)
		if err != nil {
			log.Println("Error marshalling broadcast message", err)
		}

		client := player.(Client)
		_, err = client.Conn.Write(bytes)
		if err != nil {
			log.Println("Error broadcasting to clients...", err)
			disconnectedUsers = append(disconnectedUsers, playerId.(string))

		}
		return true
	})

	for _, player := range disconnectedUsers {
		lobbyMembers.Delete(player)
	}
}
