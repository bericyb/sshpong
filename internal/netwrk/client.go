package netwrk

import (
	"fmt"
	"log"
	"net"
	"strings"

	"google.golang.org/protobuf/proto"
)

type LobbyPlayerStatus struct {
	Username string
	Status   string
}

type Interrupter struct {
	MessageType string
	Message     string
	ReplyChan   chan string
}

var username string
var lobby chan LobbyPlayerStatus
var interruptChan chan Interrupter

func ConnectToLobby(playerUsername string, messageOutputChan chan *LobbyMessage, lobbyMessageChan chan LobbyPlayerStatus, interruptChannel chan Interrupter) {
	username = playerUsername
	lobby = lobbyMessageChan
	interruptChan = interruptChannel

	conn, err := net.Dial("tcp", "127.0.0.1:12345")
	if err != nil {
		fmt.Println("Sorry, failed to connect to server...")
		return
	}

	loginMsg, err := proto.Marshal(&LobbyMessage{Type: "name", Content: username})
	if err != nil {
		fmt.Println("Sorry bro but your username is wack AF...")
	}

	_, err = conn.Write(loginMsg)
	if err != nil {
		fmt.Println("Sorry, could not communicate with server...")
	}

	fmt.Println("Starting client loop")
	messageInputChan := make(chan *LobbyMessage)
	go func() {
		for {
			messageBytes := make([]byte, 1024)

			n, err := conn.Read(messageBytes)
			if err != nil {
				fmt.Println("Sorry, failed to read message from server...", err)
			}

			message := &LobbyMessage{}

			err = proto.Unmarshal(messageBytes[:n], message)
			if err != nil {
				fmt.Println("Sorry, the server sent something weird back...")
			}

			messageInputChan <- message
		}
	}()
	for {
		select {
		case msg := <-messageInputChan:
			if isDone, gameID := handleLobbyMessage(conn, msg); isDone {
				if gameID != "" {
					interruptChan <- Interrupter{
						MessageType: "game",
						Message:     gameID,
						ReplyChan:   make(chan string),
					}
					return
				} else {
					return
				}
			}
		case msg := <-messageOutputChan:
			fmt.Println("Sending message out", msg)
			err := SendMessageToServer(conn, msg)
			if err != nil {
				fmt.Println("Error!", err)
			}
		}
	}
}

func handleLobbyMessage(serverConn net.Conn, message *LobbyMessage) (bool, string) {
	switch message.Type {
	case "text":
		fmt.Println(message.Content)
		return false, ""
	case "error":
		fmt.Println("Error:", message.Content)
		return false, ""
	case "invite":
		fmt.Println("GOT INVITE!")
		replyChan := make(chan string)
		interruptChan <- Interrupter{
			MessageType: "invite",
			Message:     fmt.Sprintf("Invite from player %s\nAccept: Y Decline: N", message.Content),
			ReplyChan:   replyChan,
		}
		input := <-replyChan
		if strings.ToLower(input) == "yes" || strings.ToLower(input) == "y" {
			SendMessageToServer(serverConn, &LobbyMessage{Type: "accept_game", Content: username})
			SendMessageToServer(serverConn, &LobbyMessage{Type: "quit", Content: username})
			return true, message.Content
		} else {
			SendMessageToServer(serverConn, &LobbyMessage{Type: "decline_game", Content: username})
		}
		return false, ""
	case "accept":
		fmt.Println(message.Content, "accepted your invite. Game starting...")
		SendMessageToServer(serverConn, &LobbyMessage{Type: "quit", Content: username})
		return true, message.Content
	case "decline_game":
		fmt.Println("Sorry,", message.Content, "declined your game invite...")
		return false, ""
	case "connect":
		lobby <- LobbyPlayerStatus{Username: message.Content, Status: "connected"}
		fmt.Println(message.Content, "connected!")
		return false, ""
	case "disconnect":
		lobby <- LobbyPlayerStatus{Username: message.Content, Status: "disconnected"}
		fmt.Println(message.Content, "disconnected!")
		return false, ""
	case "pong":
		log.Println("PoNg!")
		return false, ""
	default:
		log.Println("Got message", message)
	}
	return false, ""
}

func SendMessageToServer(connection net.Conn, message *LobbyMessage) error {
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
