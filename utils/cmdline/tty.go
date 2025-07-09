package cmdline

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"slices"
	"strings"
	"syscall"

	"dlsh/utils/ansi"
	ds "dlsh/utils/datastruct"
	key "dlsh/utils/keys"
	"golang.org/x/sys/unix"
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

type CliHistory struct {
	trie  *ds.Trie
	buf   string
	index uint
	size  uint
	base  uint
}

type Tty struct {
	Prompt   string
	Inp      *Input
	Cur      *Cursor
	hist     *CliHistory
	lineIdx  uint
	sugg     *ds.Heap[*ds.TrieNode]
	oldState *term.State
	err      error
	dimX     int
	dimY     int
}

// Ioctl Realization: https://github.com/snabb/tcxpgrp
func TcGetpgrp(fd int) (pgrp int, err error) {
	return unix.IoctlGetInt(fd, unix.TIOCGPGRP)
}

func TcSetpgrp(fd int, pgrp int) (err error) {
	return unix.IoctlSetPointerInt(fd, unix.TIOCSPGRP, pgrp)
}

func IsForeground() bool {
	fd, err := unix.Open("/dev/tty", 0666, unix.O_RDONLY)
	if err != nil {
		return false
	}
	defer unix.Close(fd)

	pgrp1, err := TcGetpgrp(fd)
	if err != nil {
		return false
	}
	pgrp2 := unix.Getpgrp()
	return pgrp1 == pgrp2
}

func SigIgn() {
	signal.Ignore(syscall.SIGTTOU)
	signal.Ignore(syscall.SIGTTIN)
	signal.Ignore(syscall.SIGTSTP)
}

func SigDfl() {
	signal.Reset(syscall.SIGTSTP)
	signal.Reset(syscall.SIGTTIN)
	signal.Reset(syscall.SIGTTOU)
}

func NewCliHistory() *CliHistory {
	ptr := new(CliHistory)
	ptr.trie = ds.NewTrie()
	return ptr
}

func (hist *CliHistory) Append(line string) {
	hist.trie.Insert(line)
	hist.size = uint(hist.trie.Size())
	hist.index = hist.size
}

func (hist *CliHistory) PrevLine() (string, error) {
	if hist.index > hist.size || hist.size == 0 {
		return "", fmt.Errorf("Invalid index: %d", hist.index)
	}
	prevNode := hist.trie.NodeAt(hist.index)
	if hist.index != 0 {
		hist.index--
		if prevNode == hist.trie.NodeAt(hist.index) {
			return hist.PrevLine()
		}
	}
	return hist.trie.At(hist.index)
}

func (hist *CliHistory) NextLine() (string, error) {
	if hist.index > hist.size || hist.size == 0 {
		return "", fmt.Errorf("Invalid index: %d", hist.index)
	}
	prevNode := hist.trie.NodeAt(hist.index)
	if hist.index < hist.size {
		hist.index++
		if prevNode == hist.trie.NodeAt(hist.index) {
			return hist.NextLine()
		}
	}
	return hist.trie.At(hist.index)
}

func (hist *CliHistory) LoadHist() {
	fp, err := os.OpenFile(os.Getenv("HOME")+"/.dlshrc", os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Unable to open ~/.dlshrc")
		return
	}
	defer fp.Close()

	scanner := bufio.NewScanner(fp)
	for scanner.Scan() {
		hist.Append(scanner.Text())
	}
	if err = scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Scanner err: %s", err.Error())
	}
	hist.base = hist.size
	hist.index = hist.base
}

func (hist *CliHistory) DumpHist() {
	fp, err := os.OpenFile(os.Getenv("HOME")+"/.dlshrc", os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Unable to open ~/.dlshrc: ", err.Error())
		return
	}
	defer fp.Close()

	writer := bufio.NewWriter(fp)
	for i := hist.base; i < hist.size; i++ {
		line, err := hist.trie.At(i)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s", err.Error())
			return
		}
		n, err := writer.WriteString(line + "\n")
		if err != nil {
			fmt.Fprintf(
				os.Stderr, "Unable to write to ~/.dlshrc %d / %d : %s", n, len(line)+1, err.Error(),
			)
			return
		}
	}
	writer.Flush()
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
	tty.oldState, tty.err = term.GetState(int(os.Stdin.Fd()))
	if tty.err != nil {
		fmt.Println(tty.err)
		os.Exit(1)
	}
	tty.dimX, tty.dimY, _ = GetTermSize()
	return tty
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
	fmt.Print(
		ansi.Dim + top.GetString()[tty.Inp.Len():] + ansi.Reset,
	)
}

func (tty *Tty) Read() string {
	tty.Reset()
	tty.dimX, tty.dimY, _ = GetTermSize()
	input := tty.Inp
	cursor := tty.Cur
	exit := false
	var rowOffset, colOffset uint
	var rowBuffr uint
	for {
		lineSize := uint(tty.dimX) - cursor.col
		rowBuffr = uint(input.Len()) / lineSize
		for row := range rowBuffr + 1 {
			cursor.ReflectPosOffsetRow(row)
			tty.ClearLine(CursorToEnd)
			start := row * lineSize
			end := min(start+lineSize, uint(len(input.line)))
			fmt.Printf("%s", input.line[start:end])
		}
		tty.Suggest()

		rowOffset = input.index / lineSize
		colOffset = input.index % lineSize
		cursor.ReflectPosOffset(rowOffset, colOffset)
		cursor.Block()

		n, err := os.Stdin.Read(input.b[:])
		if err != nil {
			fmt.Println("DED\r\n")
		}
		if n > 2 && input.b[0] == key.Escape && input.b[1] == key.OpenSqBracket {
			tty.HandleArrowKeys()
			continue
		}

		switch input.b[0] {
		case key.Enter:
			input.Str()
			exit = true
		case key.Backspace:
			if input.Len() > 0 && input.index > 0 {
				input.line = slices.Delete(input.line, int(input.index)-1, int(input.index))
				input.index--
			}
		default:
			input.line = slices.Insert(input.line, int(input.index), input.b[0])
			input.index++
		}

		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		if exit {
			break
		}

		tty.sugg = tty.hist.trie.Search(string(input.line))
	}

	cursor.ReflectPos()
	tty.ClearLine(CursorToEnd)
	fmt.Printf("%s\r\n", input.line)
	tty.hist.Append(input.str)
	tty.lineIdx++
	return input.str
}

func (tty *Tty) HandleArrowKeys() {
	input := tty.Inp
	switch input.b[2] {
	case key.Up:
		hist := tty.hist
		if hist.size == 0 {
			break
		}
		if hist.index-hist.base == tty.lineIdx {
			hist.buf = input.Str()
		}

		var pline string
		if tty.sugg != nil && tty.sugg.Size() > 0 {
			if !tty.sugg.HasNext() {
				break
			}
			tty.sugg.Next()
			top, _ := tty.sugg.Top()
			hist.index, _ = tty.sugg.TopPriority()
			pline = top.GetString()
		} else {
			if hist.index == hist.size && input.Len() != 0 {
				break
			}
			pline, _ = hist.PrevLine()
		}
		input.line = []byte(pline)
		input.index = uint(input.Len())

	case key.Down:
		hist := tty.hist
		if hist.size == 0 || hist.index == hist.size {
			break
		}

		var nline string
		if tty.sugg != nil && tty.sugg.Size() > 0 {
			if tty.sugg.HasPrev() {
				tty.sugg.Prev()
				top, _ := tty.sugg.Top()
				hist.index, _ = tty.sugg.TopPriority()
				nline = top.GetString()
			} else {
				nline = hist.buf
				hist.index = hist.size
			}
		} else if hist.index-hist.base == tty.lineIdx-1 {
			hist.index++
		} else {
			nline, _ = hist.NextLine()
		}
		input.line = []byte(nline)
		input.index = uint(input.Len())

	case key.Left:
		fmt.Print(ansi.Left)
		if input.index > 0 {
			input.index--
		}

	case key.Right:
		fmt.Print(ansi.Right)
		if input.index == uint(input.Len()) &&
			tty.sugg != nil && tty.sugg.Size() > 0 {
			top, _ := tty.sugg.Top()
			input.line = []byte(top.GetString())
			input.index = uint(input.Len())
		} else if input.index < uint(input.Len()) {
			input.index++
		}
	}
}

func (tty *Tty) ClearLine(cl ClearLineMethod) {
	fmt.Printf("%s[%dK", ansi.Esc, cl)
}

func (tty *Tty) GetPrompt() {
	home := os.Getenv("HOME")
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Println("Failed to get current dir")
		os.Exit(1)
	}
	if strings.Index(cwd, home) == 0 {
		cwd = strings.Replace(cwd, home, "~", 1)
	}
	tty.Prompt = cwd + "> "
}

func (tty *Tty) ReflectPrompt() {
	fmt.Print(tty.Prompt)
}
