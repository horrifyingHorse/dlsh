package ansi

const (
	Esc         string = "\033"
	Dim         string = Esc + "[2m"
	Invert      string = Esc + "[7m"
	Reset       string = Esc + "[0m"
	Up          string = Esc + "[1A"
	Down        string = Esc + "[1B"
	Right       string = Esc + "[1C"
	Left        string = Esc + "[1D"
	Save        string = Esc + " 7"
	ClLine      string = Esc + "[0J"
	ClLineToEnd string = Esc + "[0K"
	Restore     string = Esc + " 8"
	CursorHide  string = Esc + "[?25l"
	CursorShow  string = Esc + "[?25h"
)
