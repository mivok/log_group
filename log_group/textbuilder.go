package main

import (
	"regexp"

	ui "github.com/gizak/termui"
)

type LogGroupTxBuilder struct {
	Colorize bool
}

var counts_re = regexp.MustCompile("^(\\d+)\\s")

func (tb LogGroupTxBuilder) Build(s string, fg, bg ui.Attribute) []ui.Cell {
	cs := make([]ui.Cell, len(s))
	runes := []rune(s)
	for i := range cs {
		currfg := fg
		currbg := bg
		if runes[i] == '*' && tb.Colorize {
			// Colorize any asterisks
			currfg = ui.ColorCyan
		}
		cs[i] = ui.Cell{Ch: runes[i], Fg: currfg, Bg: currbg}
	}
	if tb.Colorize {
		// Colorize counts at the beginning of a line
		if loc := counts_re.FindStringSubmatchIndex(s); loc != nil {
			for i := loc[2]; i < loc[3]; i++ {
				cs[i].Fg = ui.ColorGreen
			}
		}
	}
	return cs
}

func NewLogGroupTxBuilder() ui.TextBuilder {
	return LogGroupTxBuilder{true}
}

func init() {
	ui.DefaultTxBuilder = NewLogGroupTxBuilder()
}
