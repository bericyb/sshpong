package renderer

import (
	"fmt"
	"os"
	"sshpong/internal/ansii"
	"sshpong/internal/pong"
	"strings"
	"time"
)

var (
	targetFps            float64 = 60.0
	targetFpMilli        float64 = float64(targetFps) / 1000.0
	millisecondTimeFrame float64 = float64(1 / targetFpMilli)
	quit                 chan bool
	userInput            chan rune
	playerX              int = 10
	playerY              int = 10
)

func Render(state pong.GameState) {
	// drawScreen(state)
	fmt.Print("\033c")
	fmt.Println("Player 1", state.Player1.Pos.X, state.Player1.Pos.Y)
	fmt.Println("Player 2", state.Player2.Pos.X, state.Player2.Pos.Y)
}

func writeCheckerBoard(height int, width int, builder *strings.Builder) {
	for i := 0; i < height; i++ {
		for j := 0; j < width; j++ {
			if i%2 == 0 {
				if j%2 == 0 {
					builder.WriteString("█")
				} else {

					builder.WriteString(" ")
				}
			} else {
				if j%2 == 0 {
					builder.WriteString(" ")
				} else {
					builder.WriteString("█")
				}
			}
		}
	}
}

func drawScreen(state pong.GameState) {
	// width := 100
	// height := 50
	var builder = strings.Builder{}
	builder.WriteString(string(ansii.Screen.ClearScreen))
	ansii.DrawBox(&builder, ansii.Offset{X: int(state.Player1.Pos.X), Y: int(state.Player1.Pos.Y)}, 5, 1, ansii.Colors.Cyan)
	ansii.DrawPixelStyle(&builder, ansii.Offset{X: int(state.Player1.Pos.X), Y: int(state.Player1.Pos.Y)}, ansii.Colors.Purple)
	ansii.DrawPixelStyle(&builder, ansii.Offset{X: int(state.Player1.Pos.X), Y: int(state.Player1.Pos.Y) + 5}, ansii.Colors.Purple)
	ansii.DrawBox(&builder, ansii.Offset{X: int(state.Player2.Pos.X), Y: int(state.Player2.Pos.Y)}, 5, 1, ansii.Colors.Cyan)
	ansii.DrawPixelStyle(&builder, ansii.Offset{X: int(state.Player2.Pos.X), Y: int(state.Player2.Pos.Y)}, ansii.Colors.Purple)
	ansii.DrawPixelStyle(&builder, ansii.Offset{X: int(state.Player2.Pos.X), Y: int(state.Player2.Pos.Y) + 5}, ansii.Colors.Purple)
	// Quit instructions
	// builder.WriteString(string(ansii.Screen.PlaceCursor(ansii.Offset{X: 0, Y: height})))
	// builder.WriteString("q to quit")

	os.Stdout.WriteString(builder.String())
}

func drawFrameStats(frameNum int, frameTimeMs float64) {
	width, height := ansii.GetTermSize()
	var spareTimeMilli = millisecondTimeFrame - frameTimeMs
	os.Stdout.WriteString(string(ansii.Screen.PlaceCursor(ansii.Offset{X: width - 12, Y: height - 2})))
	os.Stdout.WriteString(fmt.Sprintf("Frame #: %d", frameNum))
	os.Stdout.WriteString(string(ansii.Screen.PlaceCursor(ansii.Offset{X: width - 19, Y: height - 1})))
	os.Stdout.WriteString(fmt.Sprintf("Frame Time: %.4fms", frameTimeMs))
	os.Stdout.WriteString(string(ansii.Screen.PlaceCursor(ansii.Offset{X: width - 20, Y: height})))
	os.Stdout.WriteString(fmt.Sprintf("Spare Time: %.4fms", spareTimeMilli))
}

func handleInput(rawInput rune) {
	action := ProcessInput(rawInput)
	width, height := ansii.GetTermSize()

	switch action {
	case Quit:
		fmt.Println("Quitting...")
		close(quit)
	case Left, LeftArrow:
		playerX = max(playerX-1, 0)
	case Right, RightArrow:
		playerX = min(playerX+1, width)
	case Up, UpArrow:
		playerY = max(playerY-1, 0)
	case Down, DownArrow:
		playerY = min(playerY+1, height)
	case Unknown:
	default:
		os.Stdout.WriteString(string(ansii.Screen.PlaceCursor(ansii.Offset{X: 0, Y: height - 2})))
		os.Stdout.WriteString("Unrecognized Input: " + string(action))
		close(quit)
	}
}

func waitForFpsLock(startMs float64) {
	for {
		var nowMs = float64(time.Now().UnixNano()) / 1_000_000.0
		if nowMs-startMs >= millisecondTimeFrame {
			break
		}
	}
}
