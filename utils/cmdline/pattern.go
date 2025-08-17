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
func (r *Pattern) FirstLeftOf(idx int, bfr *[]byte) int {
	slices.Reverse(*bfr)
	new_idx := max(len(*bfr)-idx, 0)
	loc := r.pattern.FindIndex((*bfr)[new_idx:])
	slices.Reverse(*bfr)
	if loc != nil {
		idx = max(idx-loc[1], 0)
	} else {
		idx = 0
	}
	return idx
}

func (r *Pattern) FirstRightOf(idx int, bfr *[]byte) int {
	loc := r.pattern.FindIndex((*bfr)[idx:])
	if loc != nil {
		idx = idx + loc[1]
	} else {
		idx = len(*bfr)
	}
	return idx
}
