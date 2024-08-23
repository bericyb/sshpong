package client

import (
	"fmt"
	"io"
	"log"
	"sshpong/internal/netwrk"
	"strings"
)

var help = fmt.Errorf("use invite <player name> to invite a player\nchat or / to send a message to the lobby\nq or quit to leave the game")

func HandleUserInput(buf []byte) (*netwrk.LobbyMessage, error) {
	input := string(buf)
	args := strings.Fields(input)
	if len(args) == 0 {
		return nil, help
	}
	switch args[0] {
	case "invite":
		if args[1] != "" {
			return &netwrk.LobbyMessage{
				Type:    "invite",
				Content: args[1],
			}, nil
		} else {
			fmt.Println("Please provide a player to invite ")
		}
	case "chat":
		if args[1] != "" {
			return &netwrk.LobbyMessage{
				Type:    "chat",
				Content: strings.Join(args[1:], " "),
			}, nil
		}
	case "/":
		if args[1] != "" {
			return &netwrk.LobbyMessage{
				Type:    "chat",
				Content: strings.Join(args[1:], " "),
			}, nil
		}
	case "quit":
		return nil, io.EOF
	case "q":
		return nil, io.EOF
	case "help":
		return nil, help
	case "h":
		return nil, help
	default:
		return nil, help
	}
	return nil, nil
}

func HandleServerMessage(message *netwrk.LobbyMessage) {
	switch message.Type {
	case "invite":
		log.Println(message.PlayerId, "is inviting you to a game.", message.Content)
	case "accepted":
		log.Println(message.PlayerId, "accepted your invite.", message.Content)
	case "text":
		log.Println(message.PlayerId, ":", message.Content)
	case "decline_game":
		log.Println("Invite was declined:", message.Content)
	case "disconnect":
		log.Println("Got disconnect for player:", message.Content)
	case "connect":
		log.Println("Got connect for player:", message.Content)
	case "pong":
		log.Println("Received", message.Content)
	default:
		log.Println("Received", message.Content)
	}
}
