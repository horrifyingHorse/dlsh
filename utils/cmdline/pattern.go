package cmdline

import (
	"regexp"
	"slices"
)

type Pattern struct {
	pattern *regexp.Regexp
}

func NewPattern(expr string) *Pattern {
	newPattern := new(Pattern)
	newPattern.pattern, _ = regexp.Compile(expr)
	return newPattern
}

// PERF: There has to be a better way

// Returns the Offset from idx (always -ve)
func (r *Pattern) FirstLeftOf(idx int, bfr []byte) int {
	slices.Reverse(bfr)
	new_idx := max(len(bfr)-idx, 0)
	loc := r.pattern.FindIndex(bfr[new_idx:])
	slices.Reverse(bfr)
	if loc != nil {
		return -loc[1]
	}
	return -idx
}
func (r *Pattern) FirstLeftIndexOf(idx int, bfr []byte) int {
	return max(idx+r.FirstLeftOf(idx, bfr), 0)
}

// Returns the Offset from idx (always +ve)
func (r *Pattern) FirstRightOf(idx int, bfr []byte) int {
	loc := r.pattern.FindIndex(bfr[idx:])
	if loc != nil {
		return loc[1]
	}
	return len(bfr) - idx
}
func (r *Pattern) FirstRightIndexOf(idx int, bfr []byte) int {
	return idx + r.FirstRightOf(idx, bfr)
}
