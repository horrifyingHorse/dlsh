package cmdline

import (
	"fmt"
	"os"

	ansi "dlsh/utils/ansi"
	key "dlsh/utils/keys"
)

type Modifier int

const (
	NoModifier   Modifier = 0
	Shift        Modifier = 50
	Alt          Modifier = 51
	ShiftAlt     Modifier = 52
	Ctrl         Modifier = 53
	CtrlShift    Modifier = 54
	CtrlAlt      Modifier = 55
	CtrlAltShift Modifier = 55
)

type Input struct {
	b         [256]byte
	finalByte byte
	bfr       []byte
	index     int
	str       string

	hasCSI   bool
	keycode  int
	modifier Modifier
}

func NewInput() *Input {
	return new(Input)
}

func (inp *Input) Reset() {
	inp.bfr = []byte{}
	inp.index = 0
	inp.str = ""
}

func (inp *Input) Len() int {
	return len(inp.bfr)
}

func (inp *Input) Str() string {
	inp.str = string(inp.bfr)
	return inp.str
}

func (inp *Input) ReadStdin() error {
	_, err := os.Stdin.Read(inp.b[:])
	if err != nil {
		fmt.Print("DED\r\n")
	}
	return err
}

func (inp *Input) ClearReadBytes() {
	for i := range inp.b {
		if inp.b[i] == 0 {
			break
		}
		inp.b[i] = 0
	}
	inp.hasCSI = false
	inp.keycode = 0
	inp.modifier = NoModifier
}

func (inp *Input) DisplayReadBytes() {
	fmt.Print("\r\n")
	fmt.Print(ansi.ClLine)
	var i int
	for i = 0; i < len(inp.b) && inp.b[i] != 0; i++ {
		fmt.Printf("%d ", inp.b[i])
	}
	fmt.Print("\r\n")
}

// Ref
// https://en.wikipedia.org/wiki/ANSI_escape_code#Terminal_input_sequences
func (inp *Input) ParseReadBytes() {
	for i := range inp.b {
		if inp.b[i] == 0 {
			break
		}
		if inp.finalByte == key.Escape && inp.b[i] == key.OpenSqBracket {
			inp.hasCSI = true
		} else if inp.hasCSI && inp.finalByte == key.OpenSqBracket {
			inp.keycode = int(inp.b[i])
		} else if inp.hasCSI && inp.finalByte == key.SemiColon {
			inp.modifier = Modifier(inp.b[i])
		}
		inp.finalByte = inp.b[i]
	}
}
