package cmdline

import (
	"fmt"
	"os"
	"slices"

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

// Line Editor
type Input struct {
	b         [256]byte
	finalByte byte
	bfr       []byte
	index     int
	str       string

	Esc      bool
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
	inp.Esc = false
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
		if inp.b[i] == key.Escape {
			inp.Esc = true
		} else if inp.finalByte == key.Escape && inp.b[i] == key.OpenSqBracket {
			inp.hasCSI = true
		} else if inp.hasCSI && inp.finalByte == key.OpenSqBracket {
			inp.keycode = int(inp.b[i])
		} else if inp.hasCSI && inp.finalByte == key.SemiColon {
			inp.modifier = Modifier(inp.b[i])
		}
		inp.finalByte = inp.b[i]
	}
}

func (inp *Input) Index() int {
	return inp.index
}

func (inp *Input) SetIndex(index int) {
	inp.index = min(max(index, 0), inp.Len())
}

func (inp *Input) SetIndexOffset(index int) {
	inp.index = min(max(inp.index+index, 0), inp.Len())
}

func (inp *Input) SetIndexMin() {
	inp.index = 0
}

func (inp *Input) SetIndexMax() {
	inp.index = inp.Len()
}

func (inp *Input) ReadEOF() bool {
	return inp.finalByte == key.CtrlD
}

// I like this but its unconventional, gotta see the diff in PERF
// NOTE: copy(dest, src) copies min(len(dest), len(src)) bytes
// keep len(dest) > 0

// func (inp *Input) Bfr(dest *[]byte) *[]byte {
// 	copy(*dest, inp.bfr)
// 	return dest
// }

// PERF: Returning a copy on heap is surely a bad idea right?
func (inp *Input) Bfr() []byte {
	buffer := make([]byte, inp.Len())
	copy(buffer, inp.bfr)
	return buffer
}

func (inp *Input) BfrDelFromTo(fromIdx, toIdx int) {
	if fromIdx < 0 {
		fromIdx = 0
	} else if fromIdx >= inp.Len() {
		fromIdx = inp.Len() - 1
	}
	if toIdx < 0 || toIdx >= inp.Len() {
		toIdx = inp.Len() - 1
	}

	if fromIdx < toIdx {
		return
	}
	inp.bfr = slices.Delete(inp.bfr, fromIdx, inp.index)
}

func (inp *Input) BfrDelCurIdxOffset(offset int) {
	newIdx := min(max(inp.index+offset, 0), inp.Len())
	if newIdx < inp.index {
		inp.bfr = slices.Delete(inp.bfr, newIdx, inp.index)
	} else {
		inp.bfr = slices.Delete(inp.bfr, inp.index, newIdx)
	}
}

func (inp *Input) BfrInsAtCurIdx(v ...byte) {
	inp.bfr = slices.Insert(inp.bfr, inp.index, v...)
}

// Sets the input bfr to string s and sets the index to len(s)
func (inp *Input) SetBfrToStr(s string) {
	inp.bfr = []byte(s)
	inp.SetIndexMax()
}
