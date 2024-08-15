package main

import (
	"bufio"
	"fmt"
	"os"
	"sshpong/internal/netwrk"
	"strings"
)

func main() {

	lobbyChan := make(chan netwrk.LobbyPlayerStatus)
	interrupter := make(chan netwrk.Interrupter)
	messageOutput := make(chan *netwrk.LobbyMessage)
	inputChan := make(chan string)

	fmt.Println("Welcome to sshpong!")
	fmt.Println("Please enter your username")

	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			text := scanner.Text()
			inputChan <- text
		}
	}()

	reader := bufio.NewReader(os.Stdin)
	username, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Error reading from your shit bro...")
	}

	go netwrk.ConnectToLobby(username, messageOutput, lobbyChan, interrupter)

	buf := make([]byte, 1024)

	for {
		select {
		case msg := <-interrupter:
			fmt.Println(msg.Message)
		default:
			n, err := os.Stdin.Read(buf)
			if err != nil {
				fmt.Println("Error reading from stdin")
				return
			}

			input := string(buf[:n])
			args := strings.Fields(input)
			switch args[0] {
			case "invite":
				if args[1] != "" {
					messageOutput <- &netwrk.LobbyMessage{
						PlayerId: username,
						Type:     "invite",
						Content:  args[1],
					}
				} else {
					fmt.Println("Please provide a player to invite ")
				}
			case "chat":
				if args[1] != "" {
					messageOutput <- &netwrk.LobbyMessage{
						PlayerId: username,
						Type:     "chat",
						Content:  strings.Join(args[1:], " "),
					}
				}
			case "/":
				if args[1] != "" {
					messageOutput <- &netwrk.LobbyMessage{
						PlayerId: username,
						Type:     "chat",
						Content:  strings.Join(args[1:], " "),
					}
				}
			case "quit":
				return
			case "q":
				return
			case "help":
				fmt.Println("use invite <player name> to invite a player\nchat or / to send a message to the lobby\nq or quit to leave the game")
			case "h":
				fmt.Println("use invite <player name> to invite a player\nchat or / to send a message to the lobby\nq or quit to leave the game")
			default:
				fmt.Println("use invite <player name> to invite a player\nchat or / to send a message to the lobby\nq or quit to leave the game")
			}

		}

	}

}
