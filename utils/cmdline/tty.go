package cmdline

// WARN: Needs urgent refactor

import (
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"slices"
	"strings"
	"sync/atomic"
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

type Tty struct {
	Prompt   string
	Inp      *Input
	Cur      *Cursor
	hist     *CliHistory
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

	tty.Cur.SetRowRelative(int(tty.Inp.index / tty.sizeX))
	tty.Cur.SetColRelative(int(tty.Inp.index % tty.sizeX))
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

	tty.Cur.SetRowRelative(int(tty.Inp.index / tty.sizeX))
	tty.Cur.SetColRelative(int(tty.Inp.index % tty.sizeX))
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

	for {
		tty.Draw()

		if exit {
			break
		}

		input.ClearReadBytes()
		_, err := os.Stdin.Read(input.b[:])
		if err != nil {
			fmt.Println("DED\r\n")
		}

		input.DisplayReadBytes()

		// https://en.wikipedia.org/wiki/ANSI_escape_code#Terminal_input_sequences
		input.ParseReadBytes()

		// is it Escape Sequecne?
		if input.hasCSI {
			// fmt.Printf("\r\nsize: %d | %d\r\n", len(input.b), input.b[6])
			// returns toContinue
			tty.HandleEscapeSequence()
			tty.HushNextSuggestion()
		} else {
			// fmt.Printf("\r\n%d\r\n", input.b[0])
			switch input.finalByte {
			case key.CtrlC:
				exit = true
				tty.NilSuggestions()
				tty.HushNextSuggestion()
			case key.CtrlD:
				// TODO: This stores "exit" in cmdhist, improve approach
				input.str = "exit"
				exit = true
				tty.NilSuggestions()
				tty.HushNextSuggestion()
			case key.Enter:
				input.Str()
				exit = true
				tty.NilSuggestions()
				tty.HushNextSuggestion()
			case key.Backspace:
				if input.Len() > 0 && input.index > 0 {
					input.bfr = slices.Delete(input.bfr, int(input.index)-1, int(input.index))
					input.index--
				}
			default:
				input.bfr = slices.Insert(input.bfr, int(input.index), input.b[0])
				input.index++
			}

			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
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

func (tty *Tty) HandleEscapeSequence() {
	input := tty.Inp
	if input.finalByte >= key.Up && input.finalByte <= key.Left {
		tty.HandleArrowKeys()
	}
	if input.finalByte == key.Tilde {
		if input.keycode == int(key.Delete) {
			if input.Len() > 0 && input.index < input.Len() {
				if input.index != input.Len()-1 {
					input.bfr = slices.Delete(input.bfr, int(input.index), int(input.index)+1)
				} else {
					input.bfr = input.bfr[:input.index]
					// Is this a good idea? I never liked Delete become backspace
					// input.index = max(input.index-1, 0)
				}
			}
		} else if input.keycode == int(key.Home) {
			input.index = 0
		} else if input.keycode == int(key.End) {
			// TODO: End also completes a suggestion
			input.index = input.Len()
		}
	}
}

func (tty *Tty) HandleArrowKeys() {
	input := tty.Inp
	switch input.finalByte {
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
		input.bfr = []byte(pline)
		input.index = input.Len()

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
		input.bfr = []byte(nline)
		input.index = input.Len()

	case key.Left:
		r, _ := regexp.Compile(`[ '"-]`)
		if input.modifier == Ctrl {
			// PERF: There has to be a better way
			slices.Reverse(input.bfr)
			new_idx := max(len(input.bfr)-input.index, 0)
			loc := r.FindIndex(input.bfr[new_idx:])
			slices.Reverse(input.bfr)
			fmt.Print("\r\n", len(loc))
			if loc != nil {
				input.index = max(input.index-loc[1], 0)
			} else {
				input.index = 0
			}
		} else {
			fmt.Print(ansi.Left) // PERF: No point at all
			if input.index > 0 {
				input.index--
			}
		}

	case key.Right:
		r, _ := regexp.Compile(`[ '"-]`)
		if input.modifier == Ctrl {
			// PERF: There has to be a better way
			loc := r.FindIndex(input.bfr[input.index:])
			if loc != nil {
				input.index = max(input.index+loc[1], 0)
			} else {
				input.index = input.Len()
			}
		} else {
			fmt.Print(ansi.Right)
			if input.index == input.Len() &&
				tty.sugg != nil && tty.sugg.Size() > 0 {
				top, _ := tty.sugg.Top()
				input.bfr = []byte(top.GetString())
				input.index = input.Len()
			} else if input.index < input.Len() {
				input.index++
			}
		}
	}
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

	// home := os.Getenv("HOME")
	// if strings.Index(cwd, home) == 0 {
	// 	cwd = strings.Replace(cwd, home, "~", 1)
	// }
	if strings.Index(cwd, "/") != -1 {
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
