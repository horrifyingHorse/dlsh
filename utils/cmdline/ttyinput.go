package cmdline

import (
	key "dlsh/utils/keys"
)

func (tty *Tty) handleInput() (bool, error) {
	input := tty.Inp

	// is it Escape Sequecne?
	if input.hasCSI {
		tty.HandleEscapeSequence()
		tty.HushNextSuggestion()
		return false, nil
	}

	exit := true
	idx := input.Index()
	switch input.finalByte {
	case key.CtrlC:
		tty.NilSuggestions()
		tty.HushNextSuggestion()
	case key.CtrlD:
		// https://unix.stackexchange.com/questions/110240/why-does-ctrl-d-eof-exit-the-shell
		// Also check the EOF behavoir to flush the stdin, top ans to :
		//			   https://stackoverflow.com/questions/1516122/how-to-capture-controld-signal
		tty.NilSuggestions()
		tty.HushNextSuggestion()
	case key.Enter:
		input.Str()
		tty.NilSuggestions()
		tty.HushNextSuggestion()
	case key.Backspace:
		exit = false
		if input.Esc {
			offset := tty.match.FirstLeftOf(idx, input.Bfr())
			input.BfrDelCurIdxOffset(offset)
			input.SetIndexOffset(offset)
		} else {
			input.BfrDelCurIdxOffset(-1)
			input.SetIndexOffset(-1)
		}
	default:
		exit = false
		input.BfrInsAtCurIdx(input.finalByte)
		input.SetIndexOffset(+1)
	}
	return exit, nil
}

func (tty *Tty) HandleEscapeSequence() {
	input := tty.Inp
	if input.finalByte >= key.Up && input.finalByte <= key.Left {
		tty.HandleArrowKeys()
	}
	if input.finalByte == key.Tilde {
		tty.HandleEscapeSequenceTerminatingTilde()
	}
}

func (tty *Tty) HandleEscapeSequenceTerminatingTilde() {
	input := tty.Inp
	idx := input.Index()
	if input.keycode == int(key.Delete) {
		if input.Len() > 0 && idx < input.Len() {
			input.BfrDelCurIdxOffset(1)
			// Is this a good idea? I never liked Delete become backspace
			// input.SetIndexOffset(-1)
		}
	} else if input.keycode == int(key.Home) {
		input.SetIndexMin()
	} else if input.keycode == int(key.End) {
		// TODO: End also completes a suggestion
		input.SetIndexMax()
	}
}

func (tty *Tty) ArrowKeyUp() {
	input := tty.Inp
	hist := tty.hist
	if hist.size == 0 {
		return
	}
	if hist.index-hist.base == tty.lineIdx {
		hist.buf = input.Str()
	}

	var pline string
	if tty.sugg != nil && tty.sugg.Size() > 0 {
		if !tty.sugg.HasNext() {
			return
		}
		tty.sugg.Next()
		top, _ := tty.sugg.Top()
		hist.index, _ = tty.sugg.TopPriority()
		pline = top.GetString()
	} else {
		if hist.index == hist.size && input.Len() != 0 {
			return
		}
		pline, _ = hist.PrevLine()
	}
	input.SetBfrToStr(pline)
}

func (tty *Tty) ArrowKeyDown() {
	input := tty.Inp
	hist := tty.hist
	if hist.size == 0 || hist.index == hist.size {
		return
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
	input.SetBfrToStr(nline)
}

func (tty *Tty) ArrowKeyLeft() {
	input := tty.Inp
	idx := input.Index()
	if input.modifier == Ctrl {
		input.SetIndexOffset(tty.match.FirstLeftOf(idx, input.Bfr()))
	} else {
		input.SetIndexOffset(-1)
	}
}

func (tty *Tty) ArrowKeyRight() {
	input := tty.Inp
	idx := input.Index()
	if input.modifier == Ctrl {
		input.SetIndexOffset(tty.match.FirstRightOf(idx, input.Bfr()))
	} else {
		if idx == input.Len() &&
			tty.sugg != nil && tty.sugg.Size() > 0 {
			top, _ := tty.sugg.Top()
			input.SetBfrToStr(top.GetString())
		} else if idx < input.Len() {
			input.SetIndexOffset(+1)
		}
	}
}

func (tty *Tty) HandleArrowKeys() {
	input := tty.Inp
	switch input.finalByte {
	case key.Up:
		tty.ArrowKeyUp()
	case key.Down:
		tty.ArrowKeyDown()
	case key.Left:
		tty.ArrowKeyLeft()
	case key.Right:
		tty.ArrowKeyRight()
	}
}
