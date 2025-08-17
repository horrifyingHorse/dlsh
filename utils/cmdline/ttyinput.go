package cmdline

import (
	"slices"

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
	switch input.finalByte {
	case key.CtrlC:
		tty.NilSuggestions()
		tty.HushNextSuggestion()
	case key.CtrlD:
		// TODO: This stores "exit" in cmdhist, improve approach
		input.str = "exit"
		tty.NilSuggestions()
		tty.HushNextSuggestion()
	case key.Enter:
		input.Str()
		tty.NilSuggestions()
		tty.HushNextSuggestion()
	case key.Backspace:
		exit = false
		if input.Esc {
			new_idx := tty.match.FirstLeftOf(input.index, &input.bfr)
			input.bfr = slices.Delete(input.bfr, new_idx, input.index)
			input.index = new_idx
		} else if input.Len() > 0 && input.index > 0 {
			input.bfr = slices.Delete(input.bfr, input.index-1, input.index)
			input.index--
		}
	default:
		exit = false
		input.bfr = slices.Insert(input.bfr, input.index, input.b[0])
		input.index++
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
	input.bfr = []byte(pline)
	input.index = input.Len()
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
	input.bfr = []byte(nline)
	input.index = input.Len()
}

func (tty *Tty) ArrowKeyLeft() {
	input := tty.Inp
	if input.modifier == Ctrl {
		input.index = tty.match.FirstLeftOf(input.index, &input.bfr)
	} else {
		if input.index > 0 {
			input.index--
		}
	}
}

func (tty *Tty) ArrowKeyRight() {
	input := tty.Inp
	if input.modifier == Ctrl {
		input.index = tty.match.FirstRightOf(input.index, &input.bfr)
	} else {
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
