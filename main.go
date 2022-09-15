package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
)

// Any text followed by a separator (symbols or whitespace)
var token_re = regexp.MustCompile(".*?([!\"#$%&'()*+,-./:;<=>?@[\\]\\\\^_`{|}~\t\n\x0b\x0c\r ]+|$)")

// Extracts a separator at the end of a token
var separator_re = regexp.MustCompile("[!\"#$%&'()*+,-./:;<=>?@[\\]\\\\^_`{|}~\t\n\x0b\x0c\r ]+$")

// Modes
const (
	MODE_LIST = iota
	MODE_DETAILS
	MODE_WILDCARD
)

type viewState struct {
	mode               int
	groups             *[][][]string
	selected_list_item int
	list_items         *[]string
}

var vs = viewState{}

// How similar log lines have to be for them to be grouped together. Expressed
// as a fraction of the number of tokens (e.g. 0.8 would be 80% of tokens must
// match)
var percent_threshold = 0.0
var reverse_sort = false
var non_interactive = false

func split_into_tokens(line string) []string {
	// Splits at whitespace or symbols. Includes the symbol at the end of each
	// token.
	return token_re.FindAllString(line, -1)
}

func matching_sections(orig, new []string) (count int) {
	// For now, we don't match if the log lines don't have the same number of
	// parts.
	if len(orig) != len(new) {
		return 0
	}
	for i := range orig {
		if orig[i] == new[i] {
			count++
		}
	}
	return
}

func process(fh *os.File) (groups [][][]string) {
	scanner := bufio.NewScanner(fh)
	for scanner.Scan() {
		best_match := -1
		best_score := 0.0
		pattern := split_into_tokens(scanner.Text())
		threshold := int(percent_threshold * float64(len(pattern)))
		for idx, patterns := range groups {
			group_pattern := patterns[0]
			match_count := matching_sections(group_pattern, pattern)
			if match_count > threshold {
				score := float64(match_count) / float64(len(pattern))
				if score > best_score {
					best_score = score
					best_match = idx
				}
			}
		}
		if best_match != -1 {
			// We have a match, append the current line to the matching group
			groups[best_match] = append(groups[best_match], pattern)
		} else {
			// Otherwise, make a new group
			groups = append(groups, [][]string{pattern})
		}
	}
	if err := scanner.Err(); err != nil {
		log.Fatal("reading file:", err)
	}
	return groups
}

func findDifferingTokens(group [][]string) []string {
	// Takes a slice of split strings, and replaces any items that differ
	// between groups with wildcards.
	wild_pattern := make([]string, len(group[0]), len(group[0]))
	copy(wild_pattern, group[0])
	for _, pattern := range group {
		for j, token := range pattern {
			if wild_pattern[j] != token {
				wild_pattern[j] = "*"
			}
		}
	}
	return wild_pattern
}

func generateWildcards(group [][]string) []string {
	wild_pattern := findDifferingTokens(group)
	// Add token separators (punctuation/spaces) back in because we just wiped
	// them out with asterisks before.
	for i, v := range wild_pattern {
		if v == "*" {
			wild_pattern[i] += separator_re.FindString(group[0][i])
		}
	}
	return wild_pattern
}

func countWildValues(group [][]string, wild_index int) []string {
	wild_pattern := findDifferingTokens(group)
	// Identify the token index with the nth wild entry
	token_index := -1
	wild_count := 0
	for i, v := range wild_pattern {
		if v == "*" {
			wild_count++
			if wild_count == wild_index {
				token_index = i
				break
			}
		}
	}
	if token_index == -1 {
		// We didn't find a matching wild pattern, so return an empty list
		return []string{}
	}
	// Now count the number of unique values in the matching token index
	group_counts := make(map[string]int)
	for _, v := range group {
		group_counts[v[token_index]]++
	}
	// Reverse the mapping so we can sort by count
	counts_group := make(map[int][]string)
	for value, count := range group_counts {
		counts_group[count] = append(counts_group[count], value)
	}
	// Sort the counts
	counts := make([]int, 0, len(counts_group))
	for k := range counts_group {
		counts = append(counts, k)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(counts)))
	// Make a list of counts -> values
	wild_counts := make([]string, 0, len(counts_group))
	for _, count := range counts {
		for _, value := range counts_group[count] {
			// Remove separators from the displayed value
			value = separator_re.ReplaceAllString(value, "")
			wild_counts = append(wild_counts, fmt.Sprintf(
				"[%-5d](fg:green) %v", count, value))
		}
	}
	return wild_counts
}

func renderGroup(group [][]string, color bool) string {
	count := len(group)
	with_wilds := generateWildcards(group)
	if color {
		return fmt.Sprintf("[%-5d](fg:green) %v", count, strings.ReplaceAll(
			strings.Join(with_wilds, ""), "*", "[*](fg:cyan)"))
	} else {
		return fmt.Sprintf("%-5d %v", count, strings.Join(with_wilds, ""))
	}
}

func switchMode(newMode int, outBox *widgets.List, param int) {
	if vs.mode == newMode {
		// Yay, nothing to do
		return
	}

	if newMode == MODE_LIST {
		outBox.Rows = *vs.list_items
		outBox.SelectedRow = vs.selected_list_item
	}

	if newMode == MODE_DETAILS && vs.mode == MODE_LIST {
		vs.selected_list_item = outBox.SelectedRow
		selected_group := (*vs.groups)[vs.selected_list_item]
		details := make([]string, 0, len(selected_group))
		for _, v := range selected_group {
			details = append(details, strings.Join(v, ""))
		}
		outBox.Rows = details
		outBox.ScrollTop()
	}

	if newMode == MODE_WILDCARD && vs.mode == MODE_LIST {
		vs.selected_list_item = outBox.SelectedRow
		selected_group := (*vs.groups)[vs.selected_list_item]

		details := countWildValues(selected_group, param)
		if len(details) == 0 {
			// We didn't find a matching group, don't switch modes
			return
		}
		outBox.Rows = details
		outBox.ScrollTop()
	}

	vs.mode = newMode
}

// Sort groups by how many log lines are in the group
type ByLength [][][]string

func (s ByLength) Len() int {
	return len(s)
}
func (s ByLength) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s ByLength) Less(i, j int) bool {
	return len(s[i]) < len(s[j])
}

func init() {
	log.SetFlags(0)
	flag.Float64Var(&percent_threshold, "threshold", 0.8,
		"Similarity threshold for log lines (0-1)")
	flag.BoolVar(&reverse_sort, "reverse", false,
		"Sort output in reverse order (non-interactive only)")
	flag.BoolVar(&non_interactive, "noninteractive", false,
		"Run in non-interactive mode (just print out grouped patterns)")
}

func main() {
	flag.Parse()
	if percent_threshold < 0.0 || percent_threshold > 1.0 {
		log.Fatal("Threshold must be between 0.0 and 1.0")
	}

	var fh *os.File
	var err error
	if flag.NArg() >= 1 {
		fh, err = os.Open(flag.Args()[0])
		if err != nil {
			log.Fatal(err)
		}
		defer fh.Close()
	} else {
		fh = os.Stdin
	}

	groups := process(fh)
	vs.groups = &groups
	if reverse_sort || !non_interactive {
		sort.Sort(sort.Reverse(ByLength(groups)))
	} else {
		sort.Sort(ByLength(groups))
	}

	// Simple printing (no interactive ui)
	if non_interactive {
		for i := range groups {
			fmt.Println(renderGroup(groups[i], false))
		}
	} else {
		err := ui.Init()
		if err != nil {
			log.Fatal(err)
		}
		defer ui.Close()

		outBox := widgets.NewList()
		outBox.Border = false
		outBox.SelectedRowStyle = ui.NewStyle(ui.ColorBlack, ui.ColorCyan)

		items := make([]string, 0, len(groups))
		for _, g := range groups {
			items = append(items, renderGroup(g, true))
		}
		vs.list_items = &items


		outBox.Rows = items
		outBox.PaddingRight = 0
		outBox.PaddingBottom = 0
		outBox.PaddingTop = 0
		outBox.PaddingLeft = 0

		helpBox := widgets.NewParagraph()
		helpBox.BorderTop = true
		helpBox.BorderBottom = false
		helpBox.BorderLeft = false
		helpBox.BorderRight = false
		helpBox.PaddingRight = 0
		helpBox.PaddingBottom = 0
		helpBox.PaddingTop = 1
		helpBox.PaddingLeft = 0
		helpBox.Text = "q:Quit  j,k,↑,↓:scroll  w:wrap  enter:details  esc:back  1-9:expand wildcard"


		termWidth, termHeight := ui.TerminalDimensions()
		outBox.SetRect(0, 0, termWidth, termHeight - 2)
		helpBox.SetRect(0, termHeight - 2, termWidth, termHeight)

		ui.Render(outBox, helpBox)

		uiEvents := ui.PollEvents()
		for {
			e := <-uiEvents
			switch e.ID {
			case "q", "<C-c>":
				return
			case "j", "<Down>":
				outBox.ScrollDown()
			case "k", "<Up>":
				outBox.ScrollUp()
			case "w":
				outBox.WrapText = !outBox.WrapText
			case "<PageDown>":
				outBox.ScrollPageDown()
			case "<PageUp>":
				outBox.ScrollPageUp()
			case "<Home>":
				outBox.ScrollTop()
			case "<End>":
				outBox.ScrollBottom()
			case "<Escape>":
				switchMode(MODE_LIST, outBox, 0)
			case "<Enter>":
				switchMode(MODE_DETAILS, outBox, 0)
			case "1", "2", "3", "4", "5", "6", "7", "8", "9":
				index, err := strconv.Atoi(e.ID)
				if err == nil {
					switchMode(MODE_WILDCARD, outBox, index)
				}
			case "<Resize>":
				termWidth, termHeight := ui.TerminalDimensions()
				outBox.SetRect(0, 0, termWidth, termHeight - 2)
				helpBox.SetRect(0, termHeight - 2, termWidth, termHeight)
			}

			ui.Render(outBox, helpBox)
		}
	}
}
