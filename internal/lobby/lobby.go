package lobby

import (
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"
	"sync"

	"github.com/google/uuid"
)

type Lobby struct {
	lobbyMembers           sync.Map
	ExternalMessageChannel chan ExternalMessage
}

type Client struct {
	Username string
	Conn     net.Conn
}

type ExternalMessage struct {
	From    string
	Target  string
	Message LobbyMessage
}

func CreateLobby() *Lobby {
	externalMessageChan := make(chan ExternalMessage)

	l := Lobby{
		lobbyMembers:           sync.Map{},
		ExternalMessageChannel: externalMessageChan,
	}

	go func(lm *sync.Map) {
		for {
			msg := <-externalMessageChan

			tc, ok := lm.Load(msg.Target)
			if !ok {
				slog.Debug("Target not found in lobby map")
				sc, ok := lm.Load(msg.From)
				if !ok {
					slog.Debug("Sender was also not found in lobby map and cannot send error message")
					continue
				}
				c, ok := sc.(Client)
				if !ok {
					slog.Debug("Item that was not a client found in the lobby map...", slog.Any("key", msg.From))
				}
				go func() {
					em := LobbyMessage{
						MessageType: "error",
						Message: Error{
							Message: fmt.Sprintf("Sorry, player %s is not available...", msg.Target),
						},
					}
					b, err := Marshal(em)
					if err != nil {
						slog.Debug("Could not marshall error message for missing player", slog.Any("error", err))
					}
					c.Conn.Write(b)
				}()
				continue
			}
			c, ok := tc.(Client)
			if !ok {

				slog.Debug("Item that was not a client found in the lobby map...", slog.Any("key", msg.From))
				continue
			}
			go func() {
				b, err := Marshal(msg.Message)
				if err != nil {
					slog.Debug("Could not marshal external message...", slog.Any("error", err))
				}
				c.Conn.Write(b)
			}()

		}
	}(&l.lobbyMembers)

	return &l
}

func (l *Lobby) HandleLobbyConnection(conn net.Conn) {
	messageBytes := make([]byte, 4096)

	ingress := make(chan LobbyMessage)
	egress := make(chan LobbyMessage)

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

			message, err = Unmarshal(messageBytes[:n])
			if err != nil {
				log.Println("Invalid message received from client", err)
			}
			ingress <- message
		}
	}()

	// Network Writer
	go func() {
		for {
			msg := <-egress
			bytes, err := Marshal(msg)
			if err != nil {
				log.Println("Error marshalling message to send to user...", err)
			}
			_, err = conn.Write(bytes)
			if err == io.EOF {
				conn.Close()
				log.Println("User has disconnected", err)

				// TODO: write message for disconnect to everyone?
				slog.Debug("Sending bad disconnect message")
				ingress <- LobbyMessage{MessageType: "disconnect", Message: Disconnect{}}
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
			serverMsg, err := l.handleClientLobbyMessage(&msg, conn)
			if err != nil {
				log.Println("Error handling client lobby message...", err)
			}
			if serverMsg.MessageType != "" {
				egress <- serverMsg
			}
		}
	}()
}

// Returns a bool of whether the player has disconnected from the lobby and an error
func (l *Lobby) handleClientLobbyMessage(message *LobbyMessage, conn net.Conn) (LobbyMessage, error) {
	switch message.MessageType {
	// Handle an name/login message from a player
	// Store the new player in the l.lobbyMembers
	// Send a connection message for each of the l.lobbyMembers to the new player
	// Send a connection message to all members in the lobby
	case "name":
		_, ok := l.lobbyMembers.Load(message.Message)
		if ok {
			return LobbyMessage{MessageType: "error", Message: Error{Message: "Sorry, that name is already taken, please try a different name"}}, nil
		}

		nm, ok := message.Message.(Name)
		if !ok {
			return LobbyMessage{MessageType: "error", Message: Error{Message: "Sorry the message value and type were not matching for name"}}, nil
		}

		l.lobbyMembers.Store(nm.Name, Client{Username: nm.Name, Conn: conn})

		// Build current lobby list
		var lobby []string
		l.lobbyMembers.Range(func(lobbyUsername any, client any) bool {
			usernameString, _ := lobbyUsername.(string)
			lobby = append(lobby, usernameString)
			return true
		})

		l.broadcastToLobby(LobbyMessage{MessageType: "connect", Message: Name{Name: nm.Name}})

		return LobbyMessage{MessageType: "name", Message: Name{
			Name: nm.Name,
		},
		}, nil

	// Handle an invite message by sending a message to the target player
	// Send an invite message to the invitee: message.Content
	// Send an ack message to the inviter: message.PlayerId
	case "invite":

		i, ok := message.Message.(Invite)
		if !ok {
			return LobbyMessage{MessageType: "error", Message: Error{Message: "Sorry the message value and type were not matching for invite"}}, nil
		}
		// TODO: figure out this shit
		l.ExternalMessageChannel <- ExternalMessage{
			From:    i.From,
			Target:  i.To,
			Message: LobbyMessage{},
		}

		return LobbyMessage{MessageType: "pending_invite", Message: PendingInvite{
			Recipient: i.To,
		}}, nil

	// Handle a accept message from a player that was invited
	// Send a game_start message back to the player: message.Content
	// Send an accepted message back to the inviter: message.PlayerId
	case "accept":
		gameID := uuid.NewString()

		am, ok := message.Message.(Accept)
		if !ok {
			return LobbyMessage{MessageType: "error", Message: Error{Message: "Sorry the message value and type were not matching for accept"}}, nil
		}

		slog.Debug("incoming accept message", slog.Any("From", am.From), slog.Any("To", am.To))
		l.ExternalMessageChannel <- ExternalMessage{
			Target: am.To,
			Message: LobbyMessage{MessageType: "accepted", Message: Accepted{
				Accepter: am.From,
				GameID:   gameID,
			},
			}}

		return LobbyMessage{MessageType: "start_game", Message: StartGame{To: am.From, GameID: gameID}}, nil
	// Handle a chat message from a player with PlayerId
	case "chat":
		c, ok := message.Message.(Chat)
		if !ok {
			return LobbyMessage{MessageType: "error", Message: Error{Message: "Sorry the message value and type were not matching for chat"}}, nil
		}
		l.broadcastToLobby(LobbyMessage{MessageType: "text", Message: Chat{
			From:    c.From,
			Message: c.Message,
		}})
		return LobbyMessage{}, nil

	// Handle a quit message from a player that was connected
	// broadcast the player quit to the lobby
	case "quit":
		q, ok := message.Message.(Disconnect)
		if !ok {
			return LobbyMessage{MessageType: "error", Message: Error{Message: "Sorry the message value and type were not matching for quit"}}, nil
		}
		l.lobbyMembers.Delete(q.From)
		l.broadcastToLobby(LobbyMessage{MessageType: "disconnect", Message: Disconnect{
			From: q.From,
		}})
		return LobbyMessage{}, nil

	// Ping and pong
	case "ping":
		return LobbyMessage{MessageType: "pong", Message: "pong"}, nil

	// Ping and pong
	default:
		return LobbyMessage{MessageType: "pong", Message: "pong"}, nil

	}
}

func (l *Lobby) broadcastToLobby(message LobbyMessage) {
	var disconnectedUsers []string
	l.lobbyMembers.Range(func(playerId, player interface{}) bool {
		bytes, err := Marshal(message)
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
		l.lobbyMembers.Delete(player)
	}
}
