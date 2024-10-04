package pong

import (
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net"
	"strings"
	"time"

	"golang.org/x/exp/rand"
)

type GameClient struct {
	Username string
	Conn     net.Conn
}

var player1 GameClient
var player2 GameClient

var ingress chan StateUpdate
var egress chan StateUpdate

const posXBound = 52
const negXBound = posXBound * -1
const posYBound = 50
const negYBound = posYBound * -1

func StartGame(conn1, conn2 net.Conn, username1, username2 string) {

	egress = make(chan StateUpdate)
	ingress = make(chan StateUpdate)

	player1 = GameClient{
		Username: username1,
		Conn:     conn1,
	}

	p1msg := StateUpdate{
		FieldPath: "isPlayer1",
		Value:     []byte{1},
	}
	b, err := json.Marshal(p1msg)
	if err != nil {
		return
	}
	_, err = player1.Conn.Write(b)
	if err != nil {
		fmt.Println("Error writing player1 msg to player1")
		return
	}

	player2 = GameClient{
		Username: username2,
		Conn:     conn2,
	}

	p2msg := StateUpdate{
		FieldPath: "isPlayer1",
		Value:     []byte{0},
	}
	b, err = json.Marshal(p2msg)
	if err != nil {
		return
	}
	_, err = player2.Conn.Write(b)
	if err != nil {
		fmt.Println("Error writing player1 msg to player2")
		return
	}

	time.Sleep(1 * time.Second)
	broadcastUpdate(StateUpdate{
		FieldPath: "Message",
		Value:     []byte("Ready..."),
	})
	time.Sleep(1 * time.Second)
	broadcastUpdate(StateUpdate{
		FieldPath: "Message",
		Value:     []byte("Set..."),
	})
	time.Sleep(1 * time.Second)
	broadcastUpdate(StateUpdate{
		FieldPath: "Message",
		Value:     []byte("Go!"),
	})
	time.Sleep(1 * time.Second)
	bv := float32(rand.Intn(2)*2 - 1)

	state := GameState{
		Score: map[string]int{player1.Username: 0, player2.Username: 0},
		Player1: Player{
			client: player1,
			Pos: Vector{
				X: -50,
				Y: 0,
			},
			Size: Vector{
				X: 1,
				Y: 10,
			},
		},
		Player2: Player{
			client: player2,
			Pos: Vector{
				X: 50,
				Y: 0,
			},
			Size: Vector{
				X: 1,
				Y: 10,
			},
		},
		Ball: Ball{
			Pos: Vector{
				X: 0,
				Y: 0,
			},
			Vel: Vector{
				X: bv,
				Y: 0,
			},
		},
	}
	go gameLoop(&state)
}

func gameLoop(state *GameState) {
	// Player 1 read loop
	go func() {
		for {

			bytes := make([]byte, 512)
			n, err := state.Player1.client.Conn.Read(bytes)
			msg := StateUpdate{}
			err = json.Unmarshal(bytes[:n], &msg)
			if err != nil {
				log.Println("error reading player 1's update request:", err)
				return
			}
			ingress <- msg
			if msg.FieldPath == "Winner" {
				return
			}
		}
	}()

	// Player 2 read loop
	go func() {
		for {

			bytes := make([]byte, 512)
			n, err := state.Player2.client.Conn.Read(bytes)
			msg := StateUpdate{}
			err = json.Unmarshal(bytes[:n], &msg)
			if err != nil {
				log.Println("error reading player 2's update request:", err)
				return
			}
			ingress <- msg
			if msg.FieldPath == "Winner" {
				return
			}
		}
	}()

	go func() {
		for {
			msg := <-egress
			broadcastUpdate(msg)
			if msg.FieldPath == "Winner" {
				return
			}
		}
	}()

	ticker := time.NewTicker(time.Second / 64)

	for {
		select {
		case msg := <-ingress:
			err := handlePlayerRequest(msg, state)
			if err != nil {
				fmt.Println("FUCK!~", err)
			}
			if msg.FieldPath == "Winner" {
				slog.Debug("Closing game loop on winner message")
				return
			}

		case _ = <-ticker.C:
			update := process(state)
			egress <- update
			if update.FieldPath == "Winner" {
				slog.Debug("Closing game loop")
				return
			}
		}
	}
}

func process(state *GameState) StateUpdate {
	// Move players
	// Check if player edge is out of bounds
	//	If out of bounds reset velocity to zero and position to edge

	if state.Player1.Pos.Y+state.Player1.Size.Y/2 > posYBound {
		state.Player1.Pos.Y = posYBound - state.Player1.Size.Y/2 - 1
	}
	if state.Player1.Pos.Y-state.Player1.Size.Y/2 < negYBound {
		state.Player1.Pos.Y = negYBound + state.Player1.Size.Y/2 + 1
	}

	if state.Player2.Pos.Y+state.Player2.Size.Y/2 > posYBound {
		state.Player2.Pos.Y = posYBound - state.Player2.Size.Y/2 - 1
	}
	if state.Player2.Pos.Y-state.Player2.Size.Y/2 < negYBound {
		state.Player2.Pos.Y = negYBound + state.Player2.Size.Y/2 + 1
	}

	// Move ball
	// Check if ball is out of bounds
	//	if out of bounds y,
	//		bounce by inverting y velocity and finding difference from bounds to out and reflect distance
	//	if out of bounds x,
	//		check if paddle is nearby, bounce by inverting and finding the remaining distance to the new position.
	//		or adjust score and ball position

	state.Ball.Pos.X = state.Ball.Pos.X + state.Ball.Vel.X
	state.Ball.Pos.Y = state.Ball.Pos.Y + state.Ball.Vel.Y

	if state.Ball.Pos.Y >= posYBound-1 && state.Ball.Vel.Y > 0 {
		state.Ball.Pos.Y = (posYBound - 1) - (state.Ball.Pos.Y - (posYBound - 1))
		state.Ball.Vel.Y = state.Ball.Vel.Y * -1
	}
	if state.Ball.Pos.Y <= negYBound+1 && state.Ball.Vel.Y < 0 {
		state.Ball.Pos.Y = (negYBound + 1) - (state.Ball.Pos.Y - (negYBound + 1))
		state.Ball.Vel.Y = state.Ball.Vel.Y * -1
	}

	// If the ball is within 1 pixel of x bounds and heading to the left (Player 1)
	if state.Ball.Pos.X <= negXBound+1 && state.Ball.Vel.X < 0 {
		// Paddle hit!
		if state.Ball.Pos.Y <= state.Player1.Pos.Y+state.Player1.Size.Y && state.Ball.Pos.Y >= state.Player1.Pos.Y-state.Player1.Size.Y {
			slog.Debug("Player1 paddle hit!")
			state.Ball.Pos.X = (negXBound + 1) - (state.Ball.Pos.X - (negXBound + 1))
			state.Ball.Vel.X = state.Ball.Vel.X * -1.001
			angleTweak := (state.Ball.Pos.Y - state.Player1.Pos.Y) / (state.Player1.Size.Y / 2)
			state.Ball.Vel.Y = angleTweak / 10
		} else {
			slog.Debug("Player1 paddle miss...")
			state.Ball.Pos.X = 0
			state.Ball.Pos.Y = 0
			state.Ball.Vel.X = 1
			state.Ball.Vel.Y = 0
			if state.Score[player2.Username] >= 9 {
				return StateUpdate{
					FieldPath: "Winner",
					Value:     []byte(player2.Username),
				}
			}
			state.Score[player2.Username] = state.Score[player2.Username] + 1
		}
	}

	// If the ball is within 1 pixel of x bounds and heading towards player2 (to the right)
	if state.Ball.Pos.X > posXBound-1 && state.Ball.Vel.X > 0 {
		// Paddle hit!
		if state.Ball.Pos.Y <= state.Player2.Pos.Y+state.Player2.Size.Y && state.Ball.Pos.Y >= state.Player2.Pos.Y-state.Player2.Size.Y {
			slog.Debug("Player2 paddle hit!")
			state.Ball.Pos.X = (posXBound - 1) - (state.Ball.Pos.X - (posXBound - 1))
			state.Ball.Vel.X = state.Ball.Vel.X * -1.001
			angleTweak := (state.Ball.Pos.Y - state.Player2.Pos.Y) / (state.Player2.Size.Y / 2)
			state.Ball.Vel.Y = angleTweak / 10
		} else {
			slog.Debug("Player2 paddle miss...")
			state.Ball.Pos.X = 0
			state.Ball.Pos.Y = 0
			state.Ball.Vel.X = -1
			state.Ball.Vel.Y = 0
			if state.Score[player1.Username] >= 9 {
				return StateUpdate{
					FieldPath: "Winner",
					Value:     []byte(player1.Username),
				}
			}
			state.Score[player1.Username] = state.Score[player1.Username] + 1
		}
	}

	ns, err := json.Marshal(state)
	if err != nil {
		slog.Debug("error marshalling entire state update", slog.Any("error", err))
	}

	return StateUpdate{
		FieldPath: "All",
		Value:     ns,
	}
}

func handlePlayerRequest(update StateUpdate, state *GameState) error {

	fields := strings.Split(update.FieldPath, ".")

	// type GameState struct {
	// 	Message string
	// 	Winner  string
	// 	Score   map[string]int
	// 	Player1 Player
	// 	Player2 Player
	// 	Ball    Ball
	// }

	switch fields[0] {
	case "Message":
		state.Message = string(update.Value)
	case "Winner":
		state.Winner = string(update.Value)
	case "Player1":
		switch fields[1] {
		case "Pos":
			v1 := Vector{}
			err := json.Unmarshal(update.Value, &v1)
			if err != nil {
				slog.Debug("error unmarshalling player1 update")
			}
			state.Player1.Pos = v1
		}
	case "Player2":
		switch fields[1] {
		case "Pos":
			v2 := Vector{}
			err := json.Unmarshal(update.Value, &v2)
			if err != nil {
				slog.Debug("error unmarshalling player2 update")
			}
			state.Player2.Pos = v2
		}
	case "Ball":
		b := Ball{}
		err := json.Unmarshal(update.Value, &b)
		if err != nil {
			slog.Debug("error unmarshalling ball update")
		}
		state.Ball = b
	}
	return nil
}

func broadcastUpdate(update StateUpdate) error {
	msg, err := json.Marshal(update)
	if err != nil {
		return err
	}
	_, err = player1.Conn.Write(msg)
	if err != nil {
		return err
	}
	_, err = player2.Conn.Write(msg)
	if err != nil {
		return err
	}

	return nil
}
