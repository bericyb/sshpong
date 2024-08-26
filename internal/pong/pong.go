package pong

import (
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"time"

	"golang.org/x/exp/rand"
	"google.golang.org/protobuf/proto"
)

type GameClient struct {
	Username string
	Conn     net.Conn
}

var player1 GameClient
var player2 GameClient

var ingress chan *ClientUpdateRequest
var egress chan *ServerUpdateMessage

const posXBound = 50
const negXBound = posXBound * -1
const posYBound = 50
const negYBound = posYBound * -1

func StartGame(conn1, conn2 net.Conn, username1, username2 string) {
	player1 = GameClient{
		Username: username1,
		Conn:     conn1,
	}
	player2 = GameClient{
		Username: username2,
		Conn:     conn2,
	}

	time.Sleep(1 * time.Second)
	broadcastUpdate(&ServerUpdateMessage{
		Type:  "message",
		Value: "Ready...",
	})
	time.Sleep(1 * time.Second)
	broadcastUpdate(&ServerUpdateMessage{
		Type:  "message",
		Value: "Set...",
	})
	time.Sleep(1 * time.Second)
	broadcastUpdate(&ServerUpdateMessage{
		Type:  "message",
		Value: "Go!",
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
			Speed: 0,
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
			Speed: 0,
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
	go gameLoop(state)
}

func gameLoop(state GameState) {
	// Player 1 read loop
	go func() {
		for {

			bytes := make([]byte, 512)
			n, err := state.Player1.client.Conn.Read(bytes)
			msg := &ClientUpdateRequest{}
			err = proto.Unmarshal(bytes[:n], msg)
			if err != nil {
				log.Println("error reading player 1's update request:", err)
				return
			}
			msg.Player = 1
			ingress <- msg
		}
	}()

	// Player 2 read loop
	go func() {
		for {

			bytes := make([]byte, 512)
			n, err := state.Player2.client.Conn.Read(bytes)
			msg := &ClientUpdateRequest{}
			err = proto.Unmarshal(bytes[:n], msg)
			if err != nil {
				log.Println("error reading player 2's update request:", err)
				return
			}
			msg.Player = 2
			ingress <- msg
		}
	}()

	go func() {
		for {
			msg := <-egress
			broadcastUpdate(msg)
		}
	}()

	ticker := time.NewTicker(time.Second / 64)

	for {
		select {
		case msg := <-ingress:
			err := handlePlayerRequest(&state, msg)
			if err != nil {
				fmt.Println("FUCK!~", err)
			}

		case _ = <-ticker.C:
			update := process(&state)
			egress <- &update
		}

	}
}

func process(state *GameState) ServerUpdateMessage {

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
			state.Ball.Pos.X = (negXBound + 1) - (state.Ball.Pos.X - (negXBound + 1))
			state.Ball.Vel.X = state.Ball.Vel.X * -1.001
			angleTweak := (state.Ball.Pos.Y - state.Player2.Pos.Y) / (state.Player2.Size.Y / 2)
			state.Ball.Vel.Y = state.Ball.Vel.Y * angleTweak
		} else {
			state.Ball.Pos.X = 0
			state.Ball.Pos.Y = 0
			state.Ball.Vel.X = 1
			state.Ball.Vel.Y = 0
			if state.Score[player2.Username] >= 9 {
				return ServerUpdateMessage{
					Type:  "gameover",
					Value: player2.Username,
				}
			}
			state.Score[player2.Username] = state.Score[player2.Username] + 1
		}
	}

	// If the ball is within 1 pixel of x bounds and heading towards player2 (to the right)
	if state.Ball.Pos.X > posXBound-1 && state.Ball.Vel.X > 0 {
		// Paddle hit!
		if state.Ball.Pos.Y <= state.Player2.Pos.Y+state.Player2.Size.Y && state.Ball.Pos.Y >= state.Player2.Pos.Y-state.Player2.Size.Y {
			state.Ball.Pos.X = (posXBound - 1) - (state.Ball.Pos.X - (posXBound - 1))
			state.Ball.Vel.X = state.Ball.Vel.X * -1.001
			angleTweak := (state.Ball.Pos.Y - state.Player2.Pos.Y) / (state.Player2.Size.Y / 2)
			state.Ball.Vel.Y = state.Ball.Vel.Y * angleTweak
		} else {
			state.Ball.Pos.X = 0
			state.Ball.Pos.Y = 0
			state.Ball.Vel.X = -1
			state.Ball.Vel.Y = 0
			if state.Score[player1.Username] >= 9 {
				return ServerUpdateMessage{
					Type:  "gameover",
					Value: player1.Username,
				}
			}
			state.Score[player1.Username] = state.Score[player1.Username] + 1
		}
	}

	return ServerUpdateMessage{}
}

func handlePlayerRequest(state *GameState, msg *ClientUpdateRequest) error {

	switch msg.Type {
	case "player_pos":
		if msg.Player == 1 {
			pos := strings.Split(msg.Value, " ")
			x, err := strconv.ParseFloat(pos[0], 32)
			if err != nil {
				fmt.Println("Got weird position update for x", err)
			}
			y, err := strconv.ParseFloat(pos[1], 32)
			if err != nil {
				fmt.Println("Got weird position update for y", err)
			}

			state.Player1.Pos = Vector{
				X: float32(x),
				Y: float32(y),
			}
		}
	default:
		fmt.Println("Got unhandled update", msg.Type)
	}

	return nil
}

func broadcastUpdate(update *ServerUpdateMessage) error {
	msg, err := proto.Marshal(update)
	if err != nil {
		return fmt.Errorf("malformed server update message %v", err)
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
