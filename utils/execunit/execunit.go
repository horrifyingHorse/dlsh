package execunit

import (
	cl "dlsh/utils/cmdline"
	ds "dlsh/utils/datastruct"
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

type ExecUnit struct {
	Piped        bool
	R, W         *os.File
	Instructions cl.Instructions
	Err          error
	Ins          *cl.Instruction
	QPid         *ds.Queue[int]
	PGrp         int
}

func NewExecUnit() *ExecUnit {
	dlsh := new(ExecUnit)
	dlsh.Piped = false
	dlsh.R = os.Stdin
	dlsh.W = os.Stdout
	dlsh.QPid = new(ds.Queue[int])
	dlsh.PGrp = unix.Getpgrp()
	return dlsh
}

func (dlsh *ExecUnit) ExecPipe() {
	ins := dlsh.Ins
	dlsh.Piped = true
	ins.PipeRead(dlsh.R)
	dlsh.R, dlsh.W, dlsh.Err = os.Pipe()
	if dlsh.Err != nil {
		fmt.Println(dlsh.Err.Error())
		os.Exit(1)
	}
	ins.PipeWrite(dlsh.W)
	if ins.IsChdir() {
		return
	}
	if dlsh.Err = ins.Cmd.Start(); dlsh.Err != nil {
		fmt.Println(dlsh.Err.Error())
		return
	}
	ins.State = true

	pid := ins.Cmd.Process.Pid
	if !cl.IsForeground() {
		dlsh.QPid.Enqueue(pid)
		return
	}

	cl.SigIgn()
	cl.TcSetpgrp(int(os.Stdin.Fd()), pid)
}

func (dlsh *ExecUnit) DrainPipeline() {
	for i, ins := range dlsh.Instructions {
		if ins.State {
			if err := ins.Cmd.Wait(); err != nil {
				fmt.Println(err.Error())
			}
			ins.State = false
			if !dlsh.QPid.Empty() {
				cl.TcSetpgrp(int(os.Stdin.Fd()), dlsh.QPid.Dequeue())
			} else {
				cl.TcSetpgrp(int(os.Stdin.Fd()), dlsh.PGrp)
				cl.SigDfl()
			}
		} else if !ins.IsChdir() {
			continue
		}
		if i > 0 && dlsh.Instructions[i-1].InsType == cl.PIPE {
			ins.R.Close()
		}
		if ins.InsType == cl.PIPE {
			ins.W.Close()
		}
	}
}

func (dlsh *ExecUnit) DrainExec() {
	ins := dlsh.Ins
	dlsh.Piped = false
	ins.PipeRead(dlsh.R)
	dlsh.R = os.Stdin
	dlsh.W = os.Stdout
	if dlsh.Err = ins.Cmd.Start(); dlsh.Err == nil {
		ins.State = true
		pid := ins.Cmd.Process.Pid
		if !cl.IsForeground() {
			dlsh.QPid.Enqueue(pid)
		} else {
			cl.SigIgn()
			cl.TcSetpgrp(int(os.Stdin.Fd()), pid)
		}
		dlsh.DrainPipeline()
	} else {
		fmt.Println(dlsh.Err.Error())
	}
}

func (dlsh *ExecUnit) Run() {
	ins := dlsh.Ins
	if err := ins.Cmd.Start(); err != nil {
		fmt.Println(err.Error())
		return
	}

	pid := ins.Cmd.Process.Pid

	cl.SigIgn()
	cl.TcSetpgrp(int(os.Stdin.Fd()), pid)
	ins.Cmd.Wait()
	cl.TcSetpgrp(int(os.Stdin.Fd()), dlsh.PGrp)
	cl.SigDfl()
}
