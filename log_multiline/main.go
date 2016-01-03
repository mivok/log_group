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

var prefix = "\\s+"
var prefix_re *regexp.Regexp

func process(fh *os.File) {
	scanner := bufio.NewScanner(fh)
	collected := make([]string, 0, 10)
	for scanner.Scan() {
		if prefix_re.MatchString(scanner.Text()) {
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
	flag.StringVar(&prefix, "prefix", "\\s+",
		"Prefix for second and successive lines of a log entry")
}

func main() {
	flag.Parse()
	var err error
	prefix_re, err = regexp.Compile("^" + prefix)
	if err != nil {
		log.Fatal("Invalid prefix regex: " + err.Error())
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
