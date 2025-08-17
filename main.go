package main

import (
	// "fmt"
	"strings"

	cl "dlsh/utils/cmdline"
	eu "dlsh/utils/execunit"
)

func main() {
	tty := cl.NewTty()
	tty.GetPrompt()
	for {
		tty.ReflectPrompt()

		tty.Raw()
		line := tty.Read()
		tty.Restore()

		line = strings.Trim(line, " \t")
		tokens := eu.Tokenize(&line)
		// fmt.Println(tokens, len(tokens))

		dlsh := eu.NewExecUnit()
		dlsh.Instructions = eu.Parse(tokens)
		for _, ins := range dlsh.Instructions {
			dlsh.Ins = ins
			cmd := ins.Cmd
			if cmd.Path == "cd" {
				if !ins.Chdir() {
					break
				}
				tty.GetPrompt()
				if ins.InsType != eu.PIPE {
					continue
				}
			} else if cmd.Path == "exit" {
				tty.DumpHist()
				return
			}

			switch ins.InsType {
			case eu.EXEC:
				if dlsh.Piped {
					dlsh.DrainExec()
				} else {
					dlsh.Run()
				}
			case eu.PIPE:
				dlsh.ExecPipe()
			case eu.WAIT:
				dlsh.DrainPipeline()
			}

			if dlsh.Err != nil {
				break
			}
		}
	}
}
