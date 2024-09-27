package client

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sshpong/internal/lobby"
	"strings"
)

type InterrupterMessage struct {
	InterruptType string
	Content       string
}

var help = fmt.Errorf("use invite <player name> to invite a player\nchat or / to send a message to the lobby\nq or quit to leave the game")

var red = "\x1b[31m"
var normal = "\033[0m"

func HandleUserInput(args []string, username string) (lobby.LobbyMessage, error) {
	if len(args) == 0 {
		return lobby.LobbyMessage{}, help
	}
	switch args[0] {
	case "invite":
		if args[1] != "" {
			return lobby.LobbyMessage{
				MessageType: "invite",
				Message:     lobby.Invite{From: username, To: args[1]}}, nil
		} else {
			fmt.Println("Please provide a player to invite ")
		}
	case "chat":
		if args[1] != "" {
			return lobby.LobbyMessage{
				MessageType: "chat",
				Message: lobby.Chat{
					From:    username,
					Message: args[1],
				},
			}, nil
		}
	case "/":
		if args[1] != "" {
			return lobby.LobbyMessage{
				MessageType: "chat",
				Message: lobby.Chat{
					From:    username,
					Message: args[1],
				},
			}, nil
		}
	case "quit":
		return lobby.LobbyMessage{}, io.EOF
	case "q":
		return lobby.LobbyMessage{}, io.EOF
	case "help":
		return lobby.LobbyMessage{}, help
	case "h":
		return lobby.LobbyMessage{}, help
	default:
		if strings.Index(args[0], "/") == 0 {
			return lobby.LobbyMessage{
				MessageType: "chat",
				Message: lobby.Chat{
					From:    username,
					Message: args[1],
				},
			}, nil
		}
		return lobby.LobbyMessage{}, help
	}
	return lobby.LobbyMessage{}, nil
}

func HandleInterruptInput(incoming InterrupterMessage, args []string, username string) (lobby.LobbyMessage, error) {

	switch incoming.InterruptType {
	case "invite":
		if len(args) < 1 {
			return lobby.LobbyMessage{
				MessageType: "decline",
				Message: lobby.Decline{
					From: username,
					To:   incoming.Content,
				},
			}, nil
		} else {
			if strings.ToLower(args[0]) == "y" || strings.ToLower(args[0]) == "yes" {
				return lobby.LobbyMessage{MessageType: "accept", Message: lobby.Accept{
					From: username,
					To:   incoming.Content,
				},
				}, nil
			}
		}

	// // Cancel waiting for invite? we aren't doing this I guess.
	// case "decline":
	// 	return nil,
	// Disconnect and connect to game
	case "accepted":
		return lobby.LobbyMessage{
			MessageType: "disconnect",
			Message: lobby.Disconnect{
				From: incoming.Content,
			},
		}, nil
	case "start_game":
		return lobby.LobbyMessage{
			MessageType: "start_game",
			Message:     lobby.StartGame{GameID: incoming.Content},
		}, nil
	}

	return lobby.LobbyMessage{}, fmt.Errorf("received a interrupt message that could not be handled %v", incoming)
}

func HandleServerMessage(message lobby.LobbyMessage) (InterrupterMessage, error) {

	msg := message.Message
	switch message.MessageType {
	case "name":

		nmsg, ok := msg.(lobby.Name)
		if !ok {
			return InterrupterMessage{}, errors.New("Not a properly formatted name message")
		}

		fmt.Printf("Current Players\n%s\n", nmsg)
	case "invite":

		imsg, ok := msg.(lobby.Invite)
		if !ok {
			return InterrupterMessage{}, errors.New("Not a propertly formatted invite message")
		}
		fmt.Println(imsg.From, "is inviting you to a game\nType y to accept...")
		return InterrupterMessage{
			InterruptType: "invite",
			Content:       imsg.From,
		}, nil
	case "pending_invite":

		pimsg, ok := msg.(lobby.PendingInvite)
		if !ok {
			return InterrupterMessage{}, errors.New("Not a properly formatted pending invite message")
		}
		fmt.Println("Invite sent to", pimsg.Recipient, "\nWaiting for response...")
	case "accepted":

		amsg, ok := msg.(lobby.Accepted)
		if !ok {
			return InterrupterMessage{}, errors.New("Not a properly formatted accepted message")
		}
		fmt.Println(amsg.Accepter, "accepted your invite.", "Press Enter to connect to game...")
		return InterrupterMessage{
			InterruptType: "start_game",
			Content:       amsg.GameID,
		}, nil
	case "start_game":

		sgmsg, ok := msg.(lobby.StartGame)
		if !ok {
			return InterrupterMessage{}, errors.New("Not a properly formatted start game message")
		}
		return InterrupterMessage{
			InterruptType: "start_game",
			Content:       sgmsg.GameID,
		}, nil
	case "chat":
		cmsg, ok := msg.(lobby.Chat)
		if !ok {
			return InterrupterMessage{}, errors.New("Not a properly formatted chat message")
		}
		fmt.Println(cmsg.From, ":", cmsg.Message)
	case "decline":
		dmsg, ok := msg.(lobby.Decline)
		if !ok {
			return InterrupterMessage{}, errors.New("Not a properly formatted decline message")
		}

		fmt.Println(dmsg.From, "declined your game invite")
	case "disconnect":

		dmsg, ok := msg.(lobby.Disconnect)
		if !ok {
			return InterrupterMessage{}, errors.New("Not a properly formatted disconnect message")
		}

		fmt.Println(dmsg.From, "has disconnected")
	case "connect":

		cmsg, ok := msg.(lobby.Connect)
		if !ok {
			return InterrupterMessage{}, errors.New("Not a properly formated connect message")
		}
		fmt.Println(cmsg.From, "has connected")
	case "pong":
		fmt.Println("Received pong")
	case "error":
		em, ok := msg.(lobby.Error)
		if !ok {
			slog.Debug("Received an indecipherable error message...", slog.Any("msg", msg))
		}
		fmt.Println(red, em.Message, normal)
	default:
		fmt.Println("Received", message.MessageType, message.Message)
	}
	return InterrupterMessage{}, nil
}
