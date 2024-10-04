package client

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sshpong/internal/lobby"
	"strings"

	"github.com/google/uuid"
)

type InterrupterMessage struct {
	InterruptType string
	Content       string
}

var help = fmt.Errorf("use invite <player name> to invite a player\nchat or / to send a message to the lobby\nq or quit to leave the game")

var Red = "\x1b[31m"
var Normal = "\033[0m"

func HandleUserInput(args []string, username string) ([]byte, error) {
	if len(args) == 0 {
		return []byte{}, help
	}
	switch args[0] {
	case "invite":
		if args[1] != "" {
			if args[1] == username {
				fmt.Println("You cannot invite yourself to a game ;)")
			} else {
				msg, err := lobby.Marshal(lobby.InviteData{From: username, To: args[1]}, lobby.Invite)
				if err != nil {
					slog.Debug("invite message was not properly marshalled", "error", err)
				}
				return msg, err
			}
		} else {
			fmt.Println("Please provide a player to invite ")
		}
	case "chat":
		if args[1] != "" {
			msg, err := lobby.Marshal(lobby.ChatData{
				From:    username,
				Message: strings.Join(args[1:], " "),
			}, lobby.Chat)
			if err != nil {
				slog.Debug("chat message was not properly marshalled", "error", err)
			}
			return msg, err
		}
	case "/":
		if args[1] != "" {
			msg, err := lobby.Marshal(lobby.ChatData{
				From:    username,
				Message: strings.Join(args[1:], " "),
			}, lobby.Chat)
			if err != nil {
				slog.Debug("chat slash message was not properly marshalled", "error", err)
			}
			return msg, err
		}
	case "quit":
		return []byte{}, io.EOF
	case "q":
		return []byte{}, io.EOF
	case "help":
		return []byte{}, help
	case "h":
		return []byte{}, help
	default:
		if strings.Index(args[0], "/") == 0 {
			msg, err := lobby.Marshal(lobby.ChatData{
				From:    username,
				Message: strings.Join(args, " ")[1:],
			}, lobby.Chat)
			if err != nil {
				slog.Debug("chat slash default message was not properly marshalled", "error", err)
			}
			return msg, err
		}
		return []byte{}, help
	}
	return []byte{}, nil
}

func HandleInterruptInput(incoming InterrupterMessage, args []string, username string) ([]byte, error) {
	switch incoming.InterruptType {
	// Respond with yes if you accept game
	case "invite":
		slog.Debug("handling invite interrupt")
		if len(args) < 1 {
			return []byte{}, nil
		} else {
			if strings.ToLower(args[0]) == "y" || strings.ToLower(args[0]) == "yes" {
				msg, err := lobby.Marshal(lobby.AcceptData{
					From:   username,
					To:     incoming.Content,
					GameID: uuid.NewString(),
				}, lobby.Accept)
				if err != nil {
					slog.Debug("accept message was not properly marshalled", "error", err)
				}
				return msg, err
			}
		}

	// TODO: Do we need this accepted? Disconnect and connect to game
	// case "accepted":
	// 	msg, err := lobby.Marshal(lobby.DisconnectData{
	// 		From: incoming.Content,
	// 	}, lobby.Disconnect)
	// 	if err != nil {
	// 		slog.Debug("disconnect message was not properly marshalled", "error", err)
	// 	}
	// 	return msg, err

	case "start_game":
		msg, err := lobby.Marshal(lobby.StartGameData{
			To:     "",
			GameID: incoming.Content,
		}, lobby.StartGame)
		if err != nil {
			slog.Debug("start game message was not properly marshalled", "error", err)
		}
		return msg, err
	}

	return []byte{}, fmt.Errorf("received a interrupt message that could not be handled %v", incoming)
}

func HandleServerMessage(msg []byte) (InterrupterMessage, error) {
	header := msg[0]
	switch header {
	case lobby.Invite:
		imsg, err := lobby.Unmarshal[lobby.InviteData](msg)
		if err != nil {
			return InterrupterMessage{}, errors.New("Not a propertly formatted invite message")
		}
		fmt.Println(imsg.From, "is inviting you to a game\nType y to accept...")
		return InterrupterMessage{
			InterruptType: "invite",
			Content:       imsg.From,
		}, nil

	case lobby.PendingInvite:
		pimsg, err := lobby.Unmarshal[lobby.PendingInviteData](msg)
		if err != nil {
			return InterrupterMessage{}, errors.New("Not a properly formatted pending invite message")
		}
		fmt.Println("Invite sent to", pimsg.Recipient, "\nWaiting for response...")

	case lobby.Accepted:
		amsg, err := lobby.Unmarshal[lobby.AcceptedData](msg)
		if err != nil {
			return InterrupterMessage{}, errors.New("Not a properly formatted accepted message")
		}
		fmt.Println(amsg.Accepter, "accepted your invite.", "Press Enter to connect to game...")
		return InterrupterMessage{
			InterruptType: "start_game",
			Content:       amsg.GameID,
		}, nil

	case lobby.StartGame:
		sgmsg, err := lobby.Unmarshal[lobby.StartGameData](msg)
		if err != nil {
			return InterrupterMessage{}, errors.New("Not a properly formatted start game message")
		}
		fmt.Println("Your invite was accepted. Press Enter to join game")
		return InterrupterMessage{
			InterruptType: "start_game",
			Content:       sgmsg.GameID,
		}, nil

	case lobby.Chat:
		cmsg, err := lobby.Unmarshal[lobby.ChatData](msg)
		if err != nil {
			return InterrupterMessage{}, errors.New("Not a properly formatted chat message")
		}
		fmt.Println(cmsg.From, ":", cmsg.Message)

	case lobby.Decline:
		dmsg, err := lobby.Unmarshal[lobby.DeclineData](msg)
		if err != nil {
			return InterrupterMessage{}, errors.New("Not a properly formatted decline message")
		}

		fmt.Println(dmsg.From, "declined your game invite")

	case lobby.Disconnect:
		dmsg, err := lobby.Unmarshal[lobby.DisconnectData](msg)
		if err != nil {
			return InterrupterMessage{}, errors.New("Not a properly formatted disconnect message")
		}
		fmt.Println(dmsg.From, "has disconnected")

	case lobby.Connect:
		cmsg, err := lobby.Unmarshal[lobby.ConnectData](msg)
		if err != nil {
			return InterrupterMessage{}, errors.New("Not a properly formated connect message")
		}
		fmt.Println(cmsg.From, "has connected")

	case lobby.CurrentlyConnected:
		ccmsg, err := lobby.Unmarshal[lobby.CurrentlyConnectedData](msg)
		if err != nil {
			return InterrupterMessage{}, errors.New("Not a properly formated connect message")
		}
		fmt.Printf("Current Players\n%s\n", ccmsg.Players)

	case lobby.Error:
		em, err := lobby.Unmarshal[lobby.ErrorData](msg)
		if err != nil {
			slog.Debug("Received an indecipherable error message...", slog.Any("msg", msg[1:]))
		}
		fmt.Println(Red, em.Message, Normal)

	}
	return InterrupterMessage{}, nil
}
