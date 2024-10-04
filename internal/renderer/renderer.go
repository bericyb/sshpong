package renderer

import (
	"fmt"
	"os"
	"sshpong/internal/ansii"
	"sshpong/internal/pong"
	"strings"

	"golang.org/x/term"
)

var (
	targetFps            float64 = 60.0
	targetFpMilli        float64 = float64(targetFps) / 1000.0
	millisecondTimeFrame float64 = float64(1 / targetFpMilli)
	quit                 chan bool
	userInput            chan rune
)

const (
	reset       string = "\033[0m"
	plain       string = ""
	bold        string = "\033[1m"
	underline   string = "\033[4m"
	black       string = "\033[30m"
	red         string = "\033[31m"
	green       string = "\033[32m"
	yellow      string = "\033[33m"
	blue        string = "\033[34m"
	purple      string = "\033[35m"
	cyan        string = "\033[36m"
	white       string = "\033[37m"
	blackBg     string = "\033[40m"
	redBg       string = "\033[41m"
	greenBg     string = "\033[42m"
	yellowBg    string = "\033[43m"
	blueBg      string = "\033[44m"
	purpleBg    string = "\033[45m"
	cyanBg      string = "\033[46m"
	whiteBg     string = "\033[47m"
	clearScreen string = "\033[2J"
	hideCursor  string = "\033[?25l"
	showCursor  string = "\033[?25h"
)

func Render(state pong.GameState) {
	// fmt.Println("Player 1", ((state.Player1.Pos.X+50)/100)*width, ((state.Player1.Pos.Y+50)/100)*height)
	// fmt.Println("Player 2", ((state.Player2.Pos.X+50)/100)*width, ((state.Player2.Pos.Y+50)/100)*height)
	// fmt.Println("Ball", ((state.Ball.Pos.X+50)/100)*width, ((state.Ball.Pos.Y+50)/100)*height)
	var builder = strings.Builder{}
	builder.WriteString(string(ansii.Screen.ClearScreen))
	x1, y1 := transformToTermPos(state.Player1.Pos)
	builder.WriteString(renderBox(&builder, x1+2, y1, 2, 10, cyan))

	x2, y2 := transformToTermPos(state.Player2.Pos)
	builder.WriteString(renderBox(&builder, x2, y2, 2, 10, purple))

	xb, yb := transformToTermPos(state.Ball.Pos)
	builder.WriteString(renderPixel(&builder, xb, yb, red))

	builder.WriteString(renderMessage(&builder, state.Message))

	os.Stdout.WriteString(builder.String())
}

func setCursorPos(x, y int) string {
	return fmt.Sprintf("\033[%d;%dH", y, x)
}

// Renders a box with center positioned at X,Y with specified width and height
func renderBox(builder *strings.Builder, X, Y, width, height int, style string) string {
	str := ""
	for x := X - (width / 2); x < X+(width/2); x++ {
		for y := Y - (height / 2); y < Y+(height/2); y++ {
			str = str + (setCursorPos(x, y) + style + "█")
		}
	}

	return str
}

func renderPixel(builder *strings.Builder, x, y int, style string) string {
	return (setCursorPos(x, y) + style + "█")
}

func renderMessage(builder *strings.Builder, message string) string {
	xm, xy := transformToTermPos(pong.Vector{X: 40, Y: 40})
	return (setCursorPos(xm, xy) + reset + message)
}

// Returns state x and y positions with center origin and 50 by 50 area
// to scaled, top-left origin coordinates for the user's terminal size.
func transformToTermPos(vec pong.Vector) (int, int) {
	iwidth, iheight, _ := term.GetSize(int(os.Stdin.Fd()))
	width := float32(iwidth)
	height := float32(iheight)

	ix := int(((vec.X + 50) / 100) * width)
	iy := int(((vec.Y + 50) / 100) * height)

	return ix, iy
}
