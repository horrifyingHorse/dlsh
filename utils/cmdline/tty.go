package cmdline

// WARN: Needs refactor

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"

	"dlsh/utils/ansi"
	ds "dlsh/utils/datastruct"

	"golang.org/x/term"
)

type ClearLineMethod int8

const (
	CursorToEnd   ClearLineMethod = 0
	CursorToStart ClearLineMethod = 1
	EntireLine    ClearLineMethod = 2
)

func GetTermSize() (int, int, error) {
	return term.GetSize(int(os.Stdin.Fd()))
}

// TODO: Separate Layout from tty: dimX, dimY, sizeX, sizeY, winch
type Tty struct {
	Prompt   string
	Inp      *Input
	Cur      *Cursor
	hist     *CliHistory
	match    *Pattern
	lineIdx  uint
	sugg     *ds.Heap[*ds.TrieNode]
	supSugg  bool
	oldState *term.State
	err      error
	dimX     int
	dimY     int
	sizeX    int
	sizeY    int

	winchDone chan bool
	sigwinch  atomic.Bool
}

func (tty *Tty) DumpHist() {
	tty.hist.DumpHist()
}

func NewTty() *Tty {
	tty := new(Tty)
	tty.Inp = NewInput()
	tty.Cur = new(Cursor)
	tty.hist = NewCliHistory()
	tty.hist.LoadHist()
	tty.lineIdx = 0
	tty.sugg = nil
	tty.supSugg = false
	tty.oldState, tty.err = term.GetState(int(os.Stdin.Fd()))
	if tty.err != nil {
		fmt.Println(tty.err)
		os.Exit(1)
	}
	tty.dimX, tty.dimY, _ = GetTermSize()
	tty.sizeX = tty.dimX - tty.Cur.initCol
	tty.sizeY = 1
	tty.winchDone = make(chan bool)
	tty.match = NewPattern(`[ '/\()"-,.|?!@#&^]`)

	return tty
}

func (tty *Tty) winch() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGWINCH)
	for {
		select {
		case <-tty.winchDone:
			signal.Stop(sig)
			close(sig)
			return
		case <-sig:
			tty.sigwinch.Store(true)
		}
	}
}

func (tty *Tty) Raw() {
	err := &tty.err
	tty.oldState, *err = term.MakeRaw(int(os.Stdin.Fd()))
	if *err != nil {
		fmt.Println(*err)
		return
	}
}

func (tty *Tty) Restore() {
	if tty.oldState == nil {
		fmt.Println("cannot restore to a nil state")
		return
	}
	term.Restore(int(os.Stdin.Fd()), tty.oldState)
}

func (tty *Tty) Reset() {
	tty.Inp.Reset()
	tty.Cur.Reset()
	tty.sugg = nil
}

func (tty *Tty) Suggest() {
	if tty.sugg == nil {
		return
	}
	var top *ds.TrieNode
	var err error
	if top, err = tty.sugg.Top(); err != nil {
		return
	}
	fmt.Print(ansi.Dim)
	tty.Append(top.GetString()[tty.Inp.Len():])
	fmt.Print(ansi.Reset)
}

func (tty *Tty) CalcLayout() (int, int) {
	x, y := GetPos()
	tty.dimX, tty.dimY, _ = GetTermSize()
	deltaX := tty.Cur.row - x
	tty.Cur.initRow -= deltaX
	deltaY := tty.Cur.col - y
	tty.Cur.initCol -= deltaY
	tty.sizeX = tty.dimX - tty.Cur.initCol
	return deltaX, deltaY
}

func (tty *Tty) CalcLayoutX() {
	tty.sizeX = tty.dimX - tty.Cur.initCol
}

func (tty *Tty) Print() {
	input := tty.Inp
	cursor := tty.Cur
	tty.sizeY = (input.Len() / tty.sizeX) + 1
	for row := range tty.sizeY {
		cursor.ReflectInitPosOffsetRow(row)
		start := row * tty.sizeX
		end := min(start+tty.sizeX, len(input.bfr))
		fmt.Printf("%s", input.bfr[start:end])
	}
}

func (tty *Tty) Append(bffr string) {
	colOffset := tty.Inp.Len() % tty.sizeX
	rowOffset := tty.Inp.Len()/tty.sizeX + tty.Cur.initRow
	rowBuffrLen := (len(bffr) + colOffset) / tty.sizeX
	offset := min(tty.sizeX-colOffset, len(bffr))
	fmt.Printf("%s", bffr[:offset])
	tty.sizeY += rowBuffrLen
	for row := range rowBuffrLen {
		tty.Cur.ReflectPosAt(rowOffset+row+1, tty.Cur.initCol)
		start := offset + row*tty.sizeX
		end := min(start+tty.sizeX, len(bffr))
		fmt.Printf("%s", bffr[start:end])
	}
}

func (tty *Tty) Clear() {
	cursor := tty.Cur
	for row := range tty.sizeY {
		cursor.ReflectInitPosOffsetRow(row)
		tty.ClearLine(CursorToEnd)
	}
}

func (tty *Tty) Draw() {
	fmt.Print(ansi.CursorHide)
	tty.Clear()
	tty.dimX, tty.dimY, _ = GetTermSize()
	// cursor.ReflectInitPos()
	tty.CalcLayoutX()
	tty.Print()
	tty.Suggest()
	fmt.Print(ansi.CursorShow)

	tty.Cur.SetRowRelative(tty.Inp.Index() / tty.sizeX)
	tty.Cur.SetColRelative(tty.Inp.Index() % tty.sizeX)
	tty.Cur.ReflectPos()
	tty.Cur.Block()
}

func (tty *Tty) DrawWinch() {
	deltaX, _ := tty.CalcLayout()
	fmt.Print(ansi.CursorHide)
	lines2clear := tty.sizeY + deltaX
	for lines2clear >= 0 {
		tty.Cur.ReflectPosAt(tty.Cur.initRow+lines2clear, 0)
		tty.ClearLine(EntireLine)
		lines2clear--
	}
	tty.Cur.ReflectPosAt(tty.Cur.initRow, 0)
	tty.ReflectPrompt()
	tty.Print()
	tty.Suggest()
	fmt.Print(ansi.CursorShow)

	tty.Cur.SetRowRelative(tty.Inp.Index() / tty.sizeX)
	tty.Cur.SetColRelative(tty.Inp.Index() % tty.sizeX)
	tty.Cur.ReflectPos()
	tty.Cur.Block()
	tty.sigwinch.Store(false)
}

func (tty *Tty) Read() string {
	// PERF: this is repetitive
	go tty.winch()

	tty.Reset()
	input := tty.Inp
	exit := false
	var err error = nil

	for {
		tty.Draw()
		if exit {
			break
		}

		input.ClearReadBytes()
		err = input.ReadStdin()
		input.DisplayReadBytes()
		input.ParseReadBytes()

		exit, _ = tty.handleInput()

		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		if tty.sigwinch.Load() {
			tty.DrawWinch()
		}
		tty.CalcSuggestions()
	}

	tty.ClearSuggestions()
	fmt.Print("\r\n")
	tty.hist.Append(input.str)
	tty.lineIdx++
	tty.winchDone <- true
	return input.str
}

func (tty *Tty) ClearLine(cl ClearLineMethod) {
	fmt.Printf("%s[%dK", ansi.Esc, cl)
}

func (tty *Tty) GetPrompt() {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Println("Failed to get current dir")
		os.Exit(1)
	}
	if strings.Contains(cwd, "/") {
		cwd = cwd[strings.LastIndex(cwd, "/")+1:]
	}
	tty.Prompt = cwd
}

func (tty *Tty) ReflectPrompt() {
	ansi.SetBgRGB(40, 44, 52)
	ansi.SetFgRGB(186, 187, 241)
	fmt.Print(ansi.BoldOn)
	fmt.Print(" " + tty.Prompt + " ")
	fmt.Print(ansi.Reset)

	ansi.SetFgRGB(186, 187, 241)
	fmt.Print(ansi.BoldOn + " ~ " + ansi.Reset)
}

func (tty *Tty) CalcSuggestions() {
	if tty.supSugg == false {
		tty.sugg = tty.hist.trie.Search(string(tty.Inp.bfr))
	}
	tty.supSugg = false
}

func (tty *Tty) NilSuggestions() {
	tty.sugg = nil
}

func (tty *Tty) ClearSuggestions() {
	tty.sugg = nil
	tty.supSugg = false
}

func (tty *Tty) HushNextSuggestion() {
	tty.supSugg = true
}
