package renderer

import (
	"fmt"
	"os"
	"sshpong/internal/ansii"
	"strings"
	"time"
	"unicode/utf8"
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

func Render() {
	now := time.Now()
	err := doFpsTest()
	if err != nil {
		fmt.Println("Error ", err)
		return
	}
	then := time.Now()

	total := float64(float64(then.UnixMicro()-now.UnixMicro()) / 1000.0)
	fmt.Printf("\n\nTook %.3f milliseconds\n", total)
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

func drawScreen(frameNum int, startMs float64) (frameTimeMs float64) {
	_ = frameNum
	_, height := ansii.GetTermSize()
	var builder = strings.Builder{}
	builder.WriteString(string(ansii.Screen.ClearScreen))
	// writeCheckerBoard(height, width, &builder)
	// var xOffset = frameNum % width
	// ansii.DrawBox(&builder, ansii.Offset{X: 0, Y: 0}, 5, 8, ansii.Colors.Purple)
	ansii.DrawBox(&builder, ansii.Offset{X: playerX, Y: playerY}, 5, 1, ansii.Colors.Cyan)
	ansii.DrawPixelStyle(&builder, ansii.Offset{X: playerX, Y: playerY}, ansii.Colors.Purple)
	ansii.DrawPixelStyle(&builder, ansii.Offset{X: playerX, Y: playerY + 5}, ansii.Colors.Purple)
	// Quit instructions
	builder.WriteString(string(ansii.Screen.PlaceCursor(ansii.Offset{X: 0, Y: height})))
	builder.WriteString("q to quit")

	os.Stdout.WriteString(builder.String())
	var frameTimeMilli = (float64(time.Now().UnixNano()) / 1_000_000.0) - startMs
	// builder.WriteString(string(ansii.Screen.Coordinate(0, 0)))
	return frameTimeMilli
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

func doFpsTest() error {
	prev, err := ansii.MakeTermRaw()
	if err != nil {
		return err
	}
	defer ansii.RestoreTerm(prev)
	quit = make(chan bool, 1)
	userInput = make(chan rune, 1)

	// User input loop
	go func() {
		for {
			buf := make([]byte, 3)
			n, err := os.Stdin.Read(buf)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error reading from stdin: %v\n", err)
				continue
			}

			if n > 0 {
				if buf[0] == 0x1b { // ESC
					if n > 1 && buf[1] == '[' { // ESC [
						switch buf[2] {
						case 'A':
							userInput <- '↑' // Up arrow
						case 'B':
							userInput <- '↓' // Down arrow
						case 'C':
							userInput <- '→' // Right arrow
						case 'D':
							userInput <- '←' // Left arrow
						default:
							userInput <- '?'
						}
					} else {
						userInput <- '?'
					}
				} else {
					r, _ := utf8.DecodeRune(buf)
					userInput <- r
				}
			}
		}
	}()
	// Rendering loop
	go func() {
		for i := 0; i <= 10_000; i++ {
			startMs := float64(time.Now().UnixNano()) / 1_000_000.0
			select {
			case <-quit:
				return
			default:
				frameTimeMs := drawScreen(i, startMs)
				drawFrameStats(i, frameTimeMs)
				waitForFpsLock(startMs)
			}
		}
		close(quit)
	}()

	os.Stdout.WriteString(string(ansii.Screen.HideCursor))
	defer os.Stdout.WriteString(string(ansii.Screen.ShowCursor))
	for {
		select {
		case <-quit:
			fmt.Println("Exiting")
			return nil
		// case ui := <-userInput:
		case input := <-userInput:
			handleInput(input)
		// fmt.Println(ui)
		default:
		}
	}
}
