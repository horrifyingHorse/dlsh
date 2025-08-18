package ansi

import (
	"fmt"
	"strconv"
)

const (
	Esc         string = "\033"
	Restore     string = Esc + " 8"
	Save        string = Esc + " 7"
	CSI         string = Esc + "["
	Dim         string = CSI + "2m"
	Invert      string = CSI + "7m"
	Reset       string = CSI + "0m"
	Up          string = CSI + "1A"
	Down        string = CSI + "1B"
	Right       string = CSI + "1C"
	Left        string = CSI + "1D"
	ClLine      string = CSI + "0J"
	ClLineToEnd string = CSI + "0K"
	CursorHide  string = CSI + "?25l"
	CursorShow  string = CSI + "?25h"
	BoldOn      string = CSI + "1m"
)

func SetBgRGB(r, g, b int) {
	fmt.Print(Esc + "[48;2;" + strconv.Itoa(r) + ";" + strconv.Itoa(g) + ";" + strconv.Itoa(b) + "m")
}

func SetFgRGB(r, g, b int) {
	fmt.Print(Esc + "[38;2;" + strconv.Itoa(r) + ";" + strconv.Itoa(g) + ";" + strconv.Itoa(b) + "m")
}
