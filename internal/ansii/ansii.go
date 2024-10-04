package ansii

import (
	"fmt"
	"log/slog"
	"math"
	"os"
	"strings"

	"golang.org/x/term"
)

type ANSI string

const (
	reset       ANSI = "\033[0m"
	plain       ANSI = ""
	bold        ANSI = "\033[1m"
	underline   ANSI = "\033[4m"
	black       ANSI = "\033[30m"
	red         ANSI = "\033[31m"
	green       ANSI = "\033[32m"
	yellow      ANSI = "\033[33m"
	blue        ANSI = "\033[34m"
	purple      ANSI = "\033[35m"
	cyan        ANSI = "\033[36m"
	white       ANSI = "\033[37m"
	blackBg     ANSI = "\033[40m"
	redBg       ANSI = "\033[41m"
	greenBg     ANSI = "\033[42m"
	yellowBg    ANSI = "\033[43m"
	blueBg      ANSI = "\033[44m"
	purpleBg    ANSI = "\033[45m"
	cyanBg      ANSI = "\033[46m"
	whiteBg     ANSI = "\033[47m"
	clearScreen ANSI = "\033[2J"
	hideCursor  ANSI = "\033[?25l"
	showCursor  ANSI = "\033[?25h"
)

type Offset struct {
	X float32
	Y float32
}

type style struct {
	Reset     ANSI
	Plain     ANSI
	Bold      ANSI
	Underline ANSI
}

type color struct {
	Black  ANSI
	Red    ANSI
	Green  ANSI
	Yellow ANSI
	Blue   ANSI
	Purple ANSI
	Cyan   ANSI
	White  ANSI
}

type screen struct {
	ClearScreen ANSI
	HideCursor  ANSI
	ShowCursor  ANSI
}

type ascii struct {
	Block string
}

func GetTermSize() (width int, height int) {
	var fd int = int(os.Stdout.Fd())
	width, height, err := term.GetSize(fd)
	if err != nil {
		fmt.Println(string(Screen.ClearScreen) + "Fatal: error getting terminal size.")
		os.Exit(1)
	}
	return width, height
}

func MakeTermRaw() (*term.State, error) {
	var fd int = int(os.Stdout.Fd())
	return term.MakeRaw(fd)
}

func RestoreTerm(prev *term.State) error {
	var fd int = int(os.Stdout.Fd())
	return term.Restore(fd, prev)
}

func (s screen) PlaceCursor(X, Y int) ANSI {
	return ANSI(fmt.Sprintf("\033[%d;%dH", Y, X))
}

var (
	Styles = style{Bold: bold, Underline: underline, Reset: reset, Plain: plain}
	Colors = color{Red: red, Green: green, Yellow: yellow, Blue: blue, Purple: purple, Cyan: cyan, White: white}
	Screen = screen{ClearScreen: clearScreen, HideCursor: hideCursor, ShowCursor: showCursor}
	Blocks = ascii{Block: "█"}
)

// Draws a box of dimensions `height` and `width` at `offset`.
// The `offset` is the top left cell of the square.
// Blocks that would be placed off screen are clipped.
func DrawBox(builder *strings.Builder, offset Offset, height int, width int, style ANSI) {
	builder.WriteString(string(style))
	for hIdx := range height {
		if hIdx == 0 || hIdx == height-1 {
			for wIdx := range width {
				drawPixel(builder, offset.X+float32(wIdx), offset.Y+float32(hIdx))
			}
		} else {
			drawPixel(builder, offset.X, offset.Y+float32(hIdx))
			drawPixel(builder, offset.X+float32(width-1), offset.Y+float32(hIdx))
		}
	}
	builder.WriteString(string(Styles.Reset))
	return
}

func drawPixel(builder *strings.Builder, offsetX, offsetY float32) {
	termWidth, termHeight := GetTermSize()

	scaledX := math.Floor(float64(offsetX * float32(termWidth)))
	scaledY := math.Floor(float64(offsetY * float32(termHeight)))

	slog.Debug("positions", slog.Any("x", scaledX), slog.Any("y", scaledY))

	// TODO: Does float to int convertion cause many problems?
	builder.WriteString(string(Screen.PlaceCursor(int(scaledX), int(scaledY)) + ANSI(Blocks.Block)))
}

func DrawPixelStyle(builder *strings.Builder, offset Offset, style ANSI) {
	builder.WriteString(string(style))
	drawPixel(builder, offset.X, offset.Y)
	builder.WriteString(string(Styles.Reset))
}
