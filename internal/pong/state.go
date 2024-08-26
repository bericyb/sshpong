package pong

type GameState struct {
	Score   map[string]int
	Player1 Player
	Player2 Player
	Ball    Ball
}

type Vector struct {
	X float32
	Y float32
}

type Player struct {
	client GameClient
	Pos    Vector
	Size   Vector
	Speed  float32
}

type Ball struct {
	Pos Vector
	Vel Vector
}
