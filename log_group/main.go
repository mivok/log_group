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

	"github.com/mivok/logtools/selection_list"

	ui "github.com/gizak/termui"
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
				"%-5d %v", count, value))
		}
	}
	return wild_counts
}

func renderGroup(group [][]string, color bool) string {
	count := len(group)
	with_wilds := generateWildcards(group)
	if color {
		return fmt.Sprintf("%-5d %v", count,
			strings.Join(with_wilds, ""))
	} else {
		return fmt.Sprintf("%-5d\t%v", count, strings.Join(with_wilds, ""))
	}
}

func switchMode(newMode int, outBox *selection_list.SelectionList, param int) {
	if vs.mode == newMode {
		// Yay, nothing to do
		return
	}

	if newMode == MODE_LIST {
		outBox.Items = *vs.list_items
		outBox.EnableSelection = true
		outBox.SelectItem(vs.selected_list_item, true)
	}

	if newMode == MODE_DETAILS && vs.mode == MODE_LIST {
		vs.selected_list_item = outBox.SelectedItem
		selected_group := (*vs.groups)[vs.selected_list_item]
		details := make([]string, 0, len(selected_group))
		for _, v := range selected_group {
			details = append(details, strings.Join(v, ""))
		}
		outBox.Items = details
		outBox.EnableSelection = false
		outBox.Scroll(0, 0, true)
	}

	if newMode == MODE_WILDCARD && vs.mode == MODE_LIST {
		vs.selected_list_item = outBox.SelectedItem
		selected_group := (*vs.groups)[vs.selected_list_item]

		details := countWildValues(selected_group, param)
		if len(details) == 0 {
			// We didn't find a matching group, don't switch modes
			return
		}
		outBox.Items = details
		outBox.EnableSelection = false
		outBox.Scroll(0, 0, true)
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

	// TODO - process multiple files
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
		outBox := selection_list.NewSelectionList()
		outBox.Border = false
		items := make([]string, 0, len(groups))
		for _, g := range groups {
			items = append(items, renderGroup(g, true))
		}
		vs.list_items = &items
		outBox.Items = items
		outBox.Height = ui.TermHeight() - 2
		helpBox := ui.NewPar("q:Quit  ^,v,<,>,pgup,pgdown,home,end:scroll  enter:details  1-9:expand wildcard")
		helpBox.Height = 2
		helpBox.BorderRight = false
		helpBox.BorderBottom = false
		helpBox.BorderLeft = false
		ui.Body.AddRows(
			ui.NewRow(ui.NewCol(12, 0, outBox)),
			ui.NewRow(ui.NewCol(12, 0, helpBox)))
		ui.Body.Align()
		ui.Render(ui.Body)

		ui.Handle("/sys/kbd/q", func(ui.Event) {
			// Quit
			ui.StopLoop()
		})

		ui.Handle("/sys/kbd/<up>", func(ui.Event) {
			// Scroll up
			outBox.SelectItem(-1, false)
			ui.Render(ui.Body)
		})

		ui.Handle("/sys/kbd/<down>", func(ui.Event) {
			// Scroll down
			outBox.SelectItem(1, false)
			ui.Render(ui.Body)
		})

		ui.Handle("/sys/kbd/<previous>", func(ui.Event) {
			// Scroll up quickly
			outBox.SelectItem(-outBox.Height, false)
			ui.Render(ui.Body)
		})

		ui.Handle("/sys/kbd/<next>", func(ui.Event) {
			// Scroll down
			outBox.SelectItem(outBox.Height, false)
			ui.Render(ui.Body)
		})

		ui.Handle("/sys/kbd/<left>", func(ui.Event) {
			// Scroll left
			outBox.Scroll(-10, 0, false)
			ui.Render(ui.Body)
		})

		ui.Handle("/sys/kbd/<right>", func(ui.Event) {
			// Scroll right
			outBox.Scroll(10, 0, false)
			ui.Render(ui.Body)
		})

		ui.Handle("/sys/kbd/<home>", func(ui.Event) {
			// Reset current view
			// Select and scroll in case we were scrolled to the right
			outBox.Scroll(0, 0, true)
			outBox.SelectItem(0, true)
			ui.Render(ui.Body)
		})

		ui.Handle("/sys/kbd/<end>", func(ui.Event) {
			// Scroll to bottom
			outBox.Scroll(0, len(outBox.Items), true)
			outBox.SelectItem(len(outBox.Items), true)
			ui.Render(ui.Body)
		})

		ui.Handle("/sys/kbd/<escape>", func(ui.Event) {
			switchMode(MODE_LIST, outBox, 0)
			ui.Render(ui.Body)
		})

		ui.Handle("/sys/kbd/<enter>", func(ui.Event) {
			switchMode(MODE_DETAILS, outBox, 0)
			ui.Render(ui.Body)
		})

		ui.Handle("/sys/kbd/", func(e ui.Event) {
			// Handle other keys - we're interested in 1-9 for wildcards
			if data, ok := e.Data.(ui.EvtKbd); ok {
				keyStr := data.KeyStr
				if len(keyStr) == 1 && keyStr <= "9" && keyStr >= "1" {
					index, err := strconv.Atoi(keyStr)
					if err == nil {
						switchMode(MODE_WILDCARD, outBox, index)
						ui.Render(ui.Body)
					}
				}
			}
		})

		ui.Handle("/sys/wnd/resize", func(e ui.Event) {
			ui.Body.Width = ui.TermWidth()
			outBox.Height = ui.TermHeight() - 3
			ui.Body.Align()
			ui.Render(ui.Body)
		})

		ui.Loop()
	}
}
