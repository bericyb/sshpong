package client

import (
	"fmt"
	"io"
	"sshpong/internal/netwrk"
	"strings"
)

type InterrupterMessage struct {
	InterruptType string
	Content       string
}

var help = fmt.Errorf("use invite <player name> to invite a player\nchat or / to send a message to the lobby\nq or quit to leave the game")

func HandleUserInput(args []string) (*netwrk.LobbyMessage, error) {
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
		if strings.Index(args[0], "/") == 0 {
			return &netwrk.LobbyMessage{
				Type:    "chat",
				Content: strings.Join(args, " ")[1:],
			}, nil
		}
		return nil, help
	}
	return nil, nil
}

func HandleInterruptInput(incoming InterrupterMessage, args []string) (*netwrk.LobbyMessage, error) {

	switch incoming.InterruptType {
	case "invite":
		if len(args) < 1 {
			return &netwrk.LobbyMessage{
				Type:    "decline",
				Content: incoming.Content,
			}, nil
		} else {
			if strings.ToLower(args[0]) == "y" || strings.ToLower(args[0]) == "yes" {
				return &netwrk.LobbyMessage{Type: "accept", Content: incoming.Content}, nil
			}
		}

	// Cancel waiting for invite?
	case "decline":

	// Disconnect and connect to game
	case "accepted":
		return &netwrk.LobbyMessage{
			Type:    "disconnect",
			Content: "",
		}, nil
	default:
		return nil, fmt.Errorf("received a interrupt message that could not be handled %v", incoming)
	}

	return nil, nil
}

func HandleServerMessage(message *netwrk.LobbyMessage) (InterrupterMessage, error) {
	switch message.Type {
	case "name":
		fmt.Printf("Current Players\n%s\n", message.Content)
	case "invite":
		fmt.Println(message.PlayerId, "is inviting you to a game\nType y to accept...")
		return InterrupterMessage{
			InterruptType: "invite",
			Content:       message.PlayerId,
		}, nil
	case "pending_invite":
		fmt.Println("Invite sent to", message.Content, "\nWaiting for response...")
	case "accepted":
		fmt.Println(message.PlayerId, "accepted your invite.\n", "Starting game...")
	case "game_start":
		fmt.Println("Invited accepted\n", "Starting game...")
	case "text":
		fmt.Println(message.PlayerId, ":", message.Content)
	case "decline_game":
		fmt.Println(message.Content, "declined your game invite")
	case "disconnect":
		fmt.Println(message.Content, "has disconnected")
	case "connect":
		fmt.Println(message.Content, "has connected")
	case "pong":
		fmt.Println("Received", message.Content)
	default:
		fmt.Println("Received", message.Content)
	}
	return InterrupterMessage{}, nil
}
