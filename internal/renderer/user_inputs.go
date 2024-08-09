package renderer

type UiAction rune

const (
	Unknown    UiAction = iota
	Quit       UiAction = 81 // 'Q'
	Left       UiAction = 65
	Up         UiAction = 87
	Right      UiAction = 68
	Down       UiAction = 83
	LeftArrow  UiAction = 8592
	UpArrow    UiAction = 8593
	RightArrow UiAction = 8594
	DownArrow  UiAction = 8595
)

func ProcessInput(rawInput rune) (action UiAction) {
	inputVal := int(rawInput)
	// Convert to UpperCase
	if inputVal >= 97 && inputVal <= 122 {
		inputVal = inputVal - 32
	}
	return UiAction(inputVal)
}
