package cmdline

import (
	"dlsh/utils/ansi"
	"errors"
	"fmt"
	"os"
	"syscall"
)

type Cursor struct {
	row, col uint
}

type Input struct {
	b     [256]byte
	line  []byte
	index uint
	str   string
}

func NewInput() *Input {
	return new(Input)
}

func (inp *Input) Reset() {
	inp.line = []byte{}
	inp.index = 0
	inp.str = ""
}

func (inp *Input) Len() int {
	return len(inp.line)
}

func (inp *Input) Str() string {
	inp.str = string(inp.line)
	return inp.str
}

func (c *Cursor) ReflectPos() {
	fmt.Printf("%s[%d;%dH", ansi.Esc, c.row, c.col)
}

func (c *Cursor) Block() {
	fmt.Print(ansi.Invert, ansi.Reset)
}

func (c *Cursor) ReflectPosOffsetCol(colOffset uint) {
	fmt.Printf("%s[%d;%dH", ansi.Esc, c.row, c.col+colOffset)
}

func (c *Cursor) ReflectPosOffsetRow(rowOffset uint) {
	fmt.Printf("%s[%d;%dH", ansi.Esc, c.row+rowOffset, c.col)
}

func (c *Cursor) ReflectPosOffset(rowOffset, colOffset uint) {
	fmt.Printf("%s[%d;%dH", ansi.Esc, c.row+rowOffset, c.col+colOffset)
}

func (c *Cursor) GetPos() error {
	fmt.Print(ansi.Esc + "[6n")

	var buf [32]byte
	n, err := os.Stdin.Read(buf[:])
	if err != nil {
		if errors.Is(err, syscall.EAGAIN) {
			fmt.Fprintln(os.Stderr, "[tty not ready for reading]")
			return err
		} else {
			fmt.Fprintf(os.Stderr, "[tty read error: %v]\n", err)
			return err
		}
	}

	_, err = fmt.Sscanf(string(buf[:n]), ansi.Esc+"[%d;%dR", &c.row, &c.col)
	if err != nil {
		return err
	}
	return nil
}

func (c *Cursor) Reset() {
	if err := c.GetPos(); err != nil {
		fmt.Println(err)
	}
}
