package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"sort"
	"strings"
)

// Any text followed by a separator (symbols or whitespace)
var token_re = regexp.MustCompile(".*?([!\"#$%&'()*+,-./:;<=>?@[\\]\\\\^_`{|}~\t\n\x0b\x0c\r ]+|$)")

// Extracts a separator at the end of a token
var separator_re = regexp.MustCompile("[!\"#$%&'()*+,-./:;<=>?@[\\]\\\\^_`{|}~\t\n\x0b\x0c\r ]+$")

// How similar log lines have to be for them to be grouped together. Expressed
// as a fraction of the number of tokens (e.g. 0.8 would be 80% of tokens must
// match)
var percent_threshold = 0.0
var reverse_sort = false

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

func generate_wildcards(group [][]string) []string {
	// Takes a slice of split strings, and replaces any matching items with
	// wildcards.
	wild_pattern := make([]string, len(group[0]), len(group[0]))
	copy(wild_pattern, group[0])
	for _, pattern := range group {
		for j, token := range pattern {
			if wild_pattern[j] != token {
				wild_pattern[j] = "*"
			}
		}
	}
	// Add token separators (punctuation/spaces) back in because we just wiped
	// them out with asterisks before.
	for i, v := range wild_pattern {
		if v == "*" {
			wild_pattern[i] += separator_re.FindString(group[0][i])
		}
	}
	return wild_pattern
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
		"Sort output in reverse order")
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
	sort.Sort(ByLength(groups))
	for i := range groups {
		if reverse_sort {
			i = len(groups) - (i + 1)
		}
		fmt.Println(len(groups[i]), "\t", strings.Join(generate_wildcards(groups[i]), ""))
	}
}
