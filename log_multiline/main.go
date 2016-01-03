package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
)

var regex = "^\\s+"
var regex_re *regexp.Regexp

func process(fh *os.File) {
	scanner := bufio.NewScanner(fh)
	collected := make([]string, 0, 10)
	for scanner.Scan() {
		if regex_re.MatchString(scanner.Text()) {
			// The log entry is part of a multiline entry
			collected = append(collected, scanner.Text())
		} else {
			// We have a new log entry
			fmt.Println(strings.Join(collected, " "))
			// Wipe out the collected lines
			collected = []string{scanner.Text()}
		}
	}
}

func init() {
	log.SetFlags(0)
	flag.StringVar(&regex, "regex", "\\s+",
		"regex to match second and successive lines of a log entry")
}

func main() {
	flag.Parse()
	var err error
	regex_re, err = regexp.Compile(regex)
	if err != nil {
		log.Fatal(err.Error())
	}

	if flag.NArg() >= 1 {
		for _, v := range flag.Args() {
			fh, err := os.Open(v)
			if err != nil {
				log.Fatal(err)
			}
			defer fh.Close()
			process(fh)
		}
	} else {
		process(os.Stdin)
	}
}
