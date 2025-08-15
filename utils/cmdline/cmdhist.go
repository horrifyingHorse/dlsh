package cmdline

import (
	"bufio"
	"fmt"
	"os"

	ds "dlsh/utils/datastruct"
)

type CliHistory struct {
	trie  *ds.Trie
	buf   string
	index uint
	size  uint
	base  uint
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
