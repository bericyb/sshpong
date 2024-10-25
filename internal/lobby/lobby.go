package lobby

import (
	"fmt"
	"log"
	"log/slog"
	"net"
	"sync"
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
	Message []byte
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
			slog.Debug("forwarding external message")

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
					b, err := Marshal(ErrorData{Message: fmt.Sprintf("Sorry player %s is not available...", msg.Target)}, Error)
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
				lm.Delete(msg.Target)
				continue
			}
			go func() {
				c.Conn.Write(msg.Message)
			}()

		}
	}(&l.lobbyMembers)

	return &l
}

func (l *Lobby) HandleLobbyConnection(client Client) {
	messageBytes := make([]byte, 4096)

	ingress := make(chan []byte)
	egress := make(chan []byte)

	// Network Reader
	go func() {
		for {
			n, err := client.Conn.Read(messageBytes)
			if err != nil {
				client.Conn.Close()
				log.Printf("Error reading message %v", err)
				l.lobbyMembers.Delete(client.Username)

				// Server receives a disconnect message of the user
				msg, err := Marshal(DisconnectData{
					From: client.Username,
				}, Disconnect)
				if err != nil {
					slog.Error("error marshalling responsive disconnect of EOF error", "error", err)
				} else {
					ingress <- msg
				}
				return
			}
			ingress <- messageBytes[:n]
		}
	}()

	// Network Writer
	go func() {
		for {
			msg := <-egress
			_, err := client.Conn.Write(msg)
			if err != nil {
				client.Conn.Close()

				l.lobbyMembers.Delete(client.Username)

				// Server receives a disconnect message of the user
				msg, err := Marshal(DisconnectData{
					From: client.Username,
				}, Disconnect)
				if err != nil {
					slog.Error("error marshalling responsive disconnect of EOF error", "error", err)
				} else {
					ingress <- msg
				}
			}
		}
	}()

	// Client message handler
	go func() {
		for {
			msg := <-ingress
			slog.Debug("Received an ingress message", "message", msg)

			resMsg, err := l.handleClientLobbyMessage(msg)
			if err != nil {
				resMsg, err = Marshal(ErrorData{
					Message: err.Error(),
				}, Error)
			}
			if len(resMsg) > 0 {
				egress <- resMsg
			}
		}
	}()
}

func (l *Lobby) handleClientLobbyMessage(msg []byte) ([]byte, error) {
	header := msg[0]

	switch header {
	case Chat:
		l.BroadcastToLobby(msg)
		return []byte{}, nil
	case Invite:
		i, err := Unmarshal[InviteData](msg)
		if err != nil {
			slog.Debug("error unmarshalling invite message", "error", err)
			return []byte{}, err
		}

		msg, err := Marshal(InviteData{
			From: i.From,
			To:   i.To,
		}, Invite)
		if err != nil {
			slog.Error("error marshalling invite data...", "error", err)
			return []byte{}, err
		}

		_, ok := l.lobbyMembers.Load(i.To)
		if !ok {
			return Marshal(ErrorData{
				Message: fmt.Sprintf("Sorry, player %s is not available.", i.To),
			}, Error)
		} else {
			l.ExternalMessageChannel <- ExternalMessage{
				From:    i.From,
				Target:  i.To,
				Message: msg,
			}
		}

		return Marshal(PendingInviteData{
			Recipient: i.To,
		}, PendingInvite)

	// TODO: is pending invite really something that we need?
	// case PendingInvite:
	// 	pi, err := Unmarshal[PendingInviteData](msg)
	// 	if err != nil {
	// 		slog.Debug("error unmarshalling pending invite message", err)
	// 		return
	// 	}

	case Accept:
		a, err := Unmarshal[AcceptData](msg)
		if err != nil {
			slog.Debug("error unmarshalling accept message", "error", err)
			return []byte{}, err
		}

		gID := a.GameID

		msg, err := Marshal(StartGameData{
			GameID: gID,
		}, StartGame)

		l.ExternalMessageChannel <- ExternalMessage{
			From:    a.From,
			Target:  a.To,
			Message: msg,
		}

		slog.Debug("Sent start game message to inviter")

		return Marshal(StartGameData{
			To:     a.From,
			From:   a.To,
			GameID: gID,
		}, StartGame)

	// TODO: figure out the accepted and start game data situation... To field is a little hard to fill.
	// case Accepted:
	// 	a, err := Unmarshal[AcceptedData](msg)
	// 	if err != nil {
	// 		slog.Debug("error unmarshalling accpeted message", "error", err)
	// 		return []byte{}, err
	// 	}
	// 	return Marshal(StartGameData{
	// 		To:     "",
	// 		GameID: a.GameID,
	// 	}, StartGame)

	// TODO: Like pending invite, I think start game is only a client message
	// case StartGame:
	// 	sg, err := Unmarshal[StartGameData](msg)
	// 	if err != nil {
	// 		slog.Debug("error unmarshalling start game message", err)
	// 		return []byte{}, err
	// 	}

	// TODO: Do we even want to support decline responses?
	// case Decline:
	// 	d, err := Unmarshal[DeclineData](msg)
	// 	if err != nil {
	// 		slog.Debug("error unmarshalling decline message", err)
	// 		return []byte{}, err
	// 	}

	case Disconnect:
		d, err := Unmarshal[DisconnectData](msg)
		if err != nil {
			slog.Debug("error unmarshalling disconnect message", "error", err)
			return []byte{}, err
		}

		l.lobbyMembers.Delete(d.From)

		msg, err := Marshal(DisconnectData{
			From: d.From,
		}, Disconnect)

		l.BroadcastToLobby(msg)

		// TODO: how do we handle a disconnect for the client's side
		return []byte{}, nil

		// TODO: This is just a client side message right...?
		// case Connect:
		// 	c, err := Unmarshal[ConnectData](msg)
		// 	if err != nil {
		// 		slog.Debug("error unmarshalling connect message", err)
		// 		return
		// 	}

		// TODO: This is just a client side message right...?
		// case Error:
		// 	e, err := Unmarshal[ErrorData](msg)
		// 	if err != nil {
		// 		slog.Debug("error unmarshalling error message", err)
		// 		return []byte{}, err
		// 	}
	}
	return []byte{}, nil
}

func (l *Lobby) BroadcastToLobby(bytes []byte) {
	var disconnectedUsers []string
	l.lobbyMembers.Range(func(playerId, player interface{}) bool {
		client := player.(Client)
		_, err := client.Conn.Write(bytes)
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
func (l *Lobby) InitialConnectionHandler(conn net.Conn) (Client, []byte, error) {
	msg := make([]byte, 256)
	nb, err := conn.Read(msg)
	if err != nil {
		slog.Debug("error reading from initial connection")
	}
	msg = msg[:nb]
	n, err := Unmarshal[NameData](msg)
	if err != nil {
		slog.Debug("error unmarshalling name message:", "error", err.Error(), "message", msg[1:nb])

		msgOut, err := Marshal(ErrorData{
			Message: "incorrectly formatted username in name message",
		}, Error)
		if err != nil {
			slog.Error("error marshalling error message for incorrectly formatted username")
		}
		return Client{}, msgOut, err
	}
	fmt.Println("Loading lobby")
	_, ok := l.lobbyMembers.Load(n.Name)
	fmt.Println("past that")
	if ok {
		msg, err := Marshal(ErrorData{
			Message: "Sorry that name is already taken, please try a different name",
		}, Error)
		if err != nil {
			slog.Error("error marshalling error on name already taken msg")
		}
		return Client{}, msg, err
	}
	h, err := Marshal(ConnectData{
		From: n.Name,
	}, Connect)
	if err != nil {
		slog.Debug("error marshalling broadcast connect message on player connect", "error", err)
		return Client{Username: n.Name, Conn: conn}, h, err
	}
	l.BroadcastToLobby(h)

	// Build current lobby list
	var lobby []string
	l.lobbyMembers.Range(func(lobbyUsername any, client any) bool {
		usernameString, _ := lobbyUsername.(string)
		lobby = append(lobby, usernameString)
		return true
	})
	msgOut, err := Marshal(CurrentlyConnectedData{Players: lobby}, CurrentlyConnected)
	if err != nil {
		slog.Debug("Error marshalling currectly connected data on player connect")
	}

	client := Client{
		Username: n.Name,
		Conn:     conn,
	}

	l.lobbyMembers.Store(n.Name, client)

	return client, msgOut, err
}
