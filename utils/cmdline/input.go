package cmdline

import (
	"dlsh/utils/ansi"
	"errors"
	"fmt"
	"os"
	"syscall"
)

type Cursor struct {
	row, col         int
	initRow, initCol int
}

type Input struct {
	b     [256]byte
	bfr   []byte
	index int
	str   string
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

func (c *Cursor) SetRowRelative(rowOffset int) {
	c.row = c.initRow + rowOffset
}

func (c *Cursor) SetColRelative(colOffset int) {
	c.col = c.initCol + colOffset
}

func (c *Cursor) ReflectPosAt(row, col int) {
	fmt.Printf("%s[%d;%dH", ansi.Esc, row, col)
}

func (c *Cursor) ReflectPos() {
	fmt.Printf("%s[%d;%dH", ansi.Esc, c.row, c.col)
}

func (c *Cursor) ReflectPosOffsetCol(colOffset int) {
	fmt.Printf("%s[%d;%dH", ansi.Esc, c.row, c.col+colOffset)
}

func (c *Cursor) ReflectPosOffsetRow(rowOffset int) {
	fmt.Printf("%s[%d;%dH", ansi.Esc, c.row+rowOffset, c.col)
}

func (c *Cursor) ReflectPosOffset(rowOffset, colOffset int) {
	fmt.Printf("%s[%d;%dH", ansi.Esc, c.row+rowOffset, c.col+colOffset)
}

func (c *Cursor) Block() {
	fmt.Print(ansi.Invert, ansi.Reset)
}

func (c *Cursor) ReflectInitPos() {
	fmt.Printf("%s[%d;%dH", ansi.Esc, c.initRow, c.initCol)
}

func (c *Cursor) ReflectInitPosOffsetCol(colOffset int) {
	fmt.Printf("%s[%d;%dH", ansi.Esc, c.initRow, c.initCol+colOffset)
}

func (c *Cursor) ReflectInitPosOffsetRow(rowOffset int) {
	fmt.Printf("%s[%d;%dH", ansi.Esc, c.initRow+rowOffset, c.initCol)
}

func (c *Cursor) ReflectInitPosOffset(rowOffset, colOffset int) {
	fmt.Printf("%s[%d;%dH", ansi.Esc, c.initRow+rowOffset, c.initCol+colOffset)
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

	_, err = fmt.Sscanf(string(buf[:n]), ansi.Esc+"[%d;%dR", &c.initRow, &c.initCol)
	if err != nil {
		return err
	}
	return nil
}

func GetPos() (int, int) {
	fmt.Print(ansi.Esc + "[6n")

	var buf [32]byte
	n, err := os.Stdin.Read(buf[:])
	if err != nil {
		if errors.Is(err, syscall.EAGAIN) {
			fmt.Fprintln(os.Stderr, "[tty not ready for reading]")
			return 0, 0
		} else {
			fmt.Fprintf(os.Stderr, "[tty read error: %v]\n", err)
			return 0, 0
		}
	}

	var x, y int
	_, err = fmt.Sscanf(string(buf[:n]), "\x1b[%d;%dR", &x, &y)
	if err != nil {
		fmt.Printf("\r\n %s, %q\r\n", err.Error(), buf)
		return 0, 0
	}
	return x, y

}

func (c *Cursor) Reset() {
	if err := c.GetPos(); err != nil {
		fmt.Println(err)
	}
	c.row, c.col = c.initRow, c.initCol
}
