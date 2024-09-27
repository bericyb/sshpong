package pong

type GameState struct {
	Message string
	Winner  string
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
}

type Ball struct {
	Pos Vector
	Vel Vector
}

type StateUpdate struct {
	// The field to update on the state object dot separated
	// I.e Player1.Speed = the speed field on Player1
	FieldPath string
	Value     []byte
}
