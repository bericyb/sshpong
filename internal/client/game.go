package client

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"os"
	"sshpong/internal/ansii"
	"sshpong/internal/pong"
	"sshpong/internal/renderer"
	"strings"
)

var state pong.GameState
var quit chan int
var egress chan pong.StateUpdate
var isPlayer1 bool = false

func Game(conn net.Conn) {
	fmt.Println("Connected to game!")

	egress = make(chan pong.StateUpdate)
	quit = make(chan int)

	// Network reader
	go func() {
		bytes := make([]byte, 512)
		for {
			n, err := conn.Read(bytes)
			if err != nil {
				slog.Debug("failed to read from game connection...")
				quit <- 1
			}
			stateUpdateHandler(bytes[:n])
		}
	}()

	// Network writer
	go func() {
		for {
			update := <-egress
			bytes, err := json.Marshal(update)
			if err != nil {
				slog.Debug("failed to unmarhal game update message from server")
			}
			_, err = conn.Write(bytes)
			if err != nil {
				slog.Debug("failed to write to game connection...")
			}
		}
	}()

	prev, err := ansii.MakeTermRaw()
	if err != nil {
		fmt.Println("Failed to make terminal raw")
		return
	}
	defer ansii.RestoreTerm(prev)

	os.Stdout.WriteString(string(ansii.Screen.HideCursor))
	defer os.Stdout.WriteString(string(ansii.Screen.ShowCursor))

	// Input handler
	go func() {
		buf := make([]byte, 3)
		for {

			n, err := os.Stdin.Read(buf)
			if err != nil {
				fmt.Println("Error reading from stdin", err)
				return
			}

			handleGameInput(buf[:n])
		}
	}()

	<-quit
	return
}

func stateUpdateHandler(bytes []byte) {
	update := pong.StateUpdate{}
	err := json.Unmarshal(bytes, &update)
	if err != nil {
		slog.Debug("error unmarshalling server json", slog.Any("Unmarshal error", err))
		update.FieldPath = "Message"
		update.Value = []byte("An error has occured ")
	}

	fields := strings.Split(update.FieldPath, ".")

	// type GameState struct {
	// 	Message string
	// 	Winner  string
	// 	Score   map[string]int
	// 	Player1 Player
	// 	Player2 Player
	// 	Ball    Ball
	// }

	// For now let's just send the whole field from a top level
	// of the state. If things are slow we can optimize that later
	val := update.Value

	switch fields[0] {
	case "All":

		ns := pong.GameState{}
		err = json.Unmarshal(val, &ns)
		if err != nil {
			slog.Debug("error unmarshalling whole state update")
			return
		}

		state = ns
	case "Message":
		state.Message = string(update.Value)
	case "Winner":
		state.Winner = string(update.Value)
	case "Score":

		sc := map[string]int{}
		err = json.Unmarshal(val, &sc)
		if err != nil {
			slog.Debug("error unmarshalling score update")
			return
		}
		state.Score = sc
	case "Player1":
		p1 := pong.Player{}
		err = json.Unmarshal(val, &p1)
		if err != nil {
			slog.Debug("error unmarshalling player1 update")
		}
		state.Player1 = p1
	case "Player2":
		p2 := pong.Player{}
		err = json.Unmarshal(val, &p2)
		if err != nil {
			slog.Debug("error unmarshalling player2 update")
		}
		state.Player2 = p2
	case "Ball":

		b := pong.Ball{}
		err = json.Unmarshal(val, &b)
		if err != nil {
			slog.Debug("error unmarshalling ball update")
		}
		state.Ball = b
	// Special update message that determines if the client is player1 or player2
	case "isPlayer1":
		if update.Value[0] != 0 {
			isPlayer1 = true
		}
	}

	renderer.Render(state)

}

func handleGameInput(bytes []byte) {
	switch bytes[0] {
	// Up
	case 'w':
		if isPlayer1 {
			state.Player1.Pos.Y = state.Player1.Pos.Y + 1
			v, err := json.Marshal(pong.Vector{
				X: 0, Y: state.Player1.Pos.Y,
			})
			if err != nil {
				slog.Debug("error marshalling player movement", slog.Any("error", err))
			}
			update := pong.StateUpdate{
				FieldPath: "Player1.Pos",
				Value:     v,
			}

			egress <- update
		} else {
			state.Player2.Pos.Y = state.Player2.Pos.Y + 1
			v, err := json.Marshal(pong.Vector{
				X: 0, Y: state.Player2.Pos.Y,
			})
			if err != nil {
				slog.Debug("error marshalling Player2 movement", slog.Any("error", err))
			}
			update := pong.StateUpdate{
				FieldPath: "Player2.Pos",
				Value:     v,
			}
			egress <- update
		}
		return
	// Down
	case 's':
		if isPlayer1 {
			state.Player1.Pos.Y = state.Player1.Pos.Y - 1
			v, err := json.Marshal(pong.Vector{
				X: 0, Y: state.Player1.Pos.Y,
			})
			if err != nil {
				slog.Debug("error marshalling player movement", slog.Any("error", err))
			}
			update := pong.StateUpdate{
				FieldPath: "Player1.Pos",
				Value:     v,
			}

			egress <- update
		} else {
			state.Player2.Pos.Y = state.Player2.Pos.Y - 1
			v, err := json.Marshal(pong.Vector{
				X: 0, Y: state.Player2.Pos.Y,
			})
			if err != nil {
				slog.Debug("error marshalling player movement", slog.Any("error", err))
			}
			update := pong.StateUpdate{
				FieldPath: "Player2.Pos",
				Value:     v,
			}
			egress <- update
		}
		return

	// Quit
	case 'q':
		// Acts as a forfeit. Other player wins
		if isPlayer1 {
			update := pong.StateUpdate{
				FieldPath: "Winner",
				Value:     []byte("Player2"),
			}
			egress <- update
		} else {
			update := pong.StateUpdate{
				FieldPath: "Winner",
				Value:     []byte("Player1"),
			}
			egress <- update
		}
		quit <- 1
		return

	// Esc char
	case 27:
		// Arrow Keys
		switch bytes[1] {
		// Up
		case 65:
			if isPlayer1 {
				state.Player1.Pos.Y = state.Player1.Pos.Y + 1
				v, err := json.Marshal(pong.Vector{
					X: 0, Y: state.Player1.Pos.Y,
				})
				if err != nil {
					slog.Debug("error marshalling player movement", slog.Any("error", err))
				}
				update := pong.StateUpdate{
					FieldPath: "Player1.Pos",
					Value:     v,
				}

				egress <- update
			} else {
				state.Player2.Pos.Y = state.Player2.Pos.Y + 1
				v, err := json.Marshal(pong.Vector{
					X: 0, Y: state.Player2.Pos.Y,
				})
				if err != nil {
					slog.Debug("error marshalling Player2 movement", slog.Any("error", err))
				}
				update := pong.StateUpdate{
					FieldPath: "Player2.Pos",
					Value:     v,
				}
				egress <- update
			}
			return
		// Down
		case 66:
			if isPlayer1 {
				state.Player1.Pos.Y = state.Player1.Pos.Y - 1
				v, err := json.Marshal(pong.Vector{
					X: 0, Y: state.Player1.Pos.Y,
				})
				if err != nil {
					slog.Debug("error marshalling player movement", slog.Any("error", err))
				}
				update := pong.StateUpdate{
					FieldPath: "Player1.Pos",
					Value:     v,
				}

				egress <- update
			} else {
				state.Player2.Pos.Y = state.Player2.Pos.Y - 1
				v, err := json.Marshal(pong.Vector{
					X: 0, Y: state.Player2.Pos.Y,
				})
				if err != nil {
					slog.Debug("error marshalling player movement", slog.Any("error", err))
				}
				update := pong.StateUpdate{
					FieldPath: "Player2.Pos",
					Value:     v,
				}
				egress <- update
			}
			return
		}
	}
}
