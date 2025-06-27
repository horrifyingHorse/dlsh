package main

import (
	"fmt"
	"os"
	"strings"

	cl "dlsh/utils/cmdline"
	eu "dlsh/utils/execunit"
)

func main() {
	fmt.Println("$HOME=" + os.Getenv("HOME"))

	tty := cl.NewTty()
	tty.GetPrompt()
	for {
		tty.ReflectPrompt()

		tty.Raw()
		line := tty.Read()
		tty.Restore()

		line = strings.Trim(line, " \t")
		tokens := cl.Tokenize(&line)
		fmt.Println(tokens, len(tokens))

		dlsh := eu.NewExecUnit()
		dlsh.Instructions = cl.Parse(tokens)
		for _, ins := range dlsh.Instructions {
			dlsh.Ins = ins
			cmd := ins.Cmd
			if cmd.Path == "cd" {
				if !ins.Chdir() {
					break
				}
				tty.GetPrompt()
				if ins.InsType != cl.PIPE {
					continue
				}
			} else if cmd.Path == "exit" {
				tty.DumpHist()
				return
			}

			switch ins.InsType {
			case cl.EXEC:
				if dlsh.Piped {
					dlsh.DrainExec()
				} else {
					dlsh.Run()
				}
			case cl.PIPE:
				dlsh.ExecPipe()
			case cl.WAIT:
				dlsh.DrainPipeline()
			}

			if dlsh.Err != nil {
				break
			}
		}
	}
}
