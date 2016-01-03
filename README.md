# Logtools

This repository contains a few simple command line tools for working with log
files. The tools are:

* log_group - identifies similar lines in a log file and groups them
  together, replacing any differences with wildcards.
* log_multiline - combine multi-line log entries into one

## Install instructions

First, get a working go environment. A simple way to do this on OSX:

    brew install go
    mkdir ~/go
    # You may want to add the following to your .bashrc
    export GOPATH=$HOME/go

Once you have a working go environment, run:

    go get github.com/mivok/logtools/...

And the tools will be installed in your GOBIN directory (`~/go/bin`) if you
used the instructions above.

## Commands

### log_group

This command will identify similar log lines and group them together,
replacing parts that vary with an asterisk.

By default, it runs in interactive mode, where you can view the groups and
drill down into the individual log lines, or view the different values for
each asterisk:

* Up/Down/Left/Right/PgUp/PgDown/Home/End - Scroll and/or select log entries
* Enter - View all log entries in a given group
* 1-9 - Expand the Nth asterisk and show what values it has in each group,
  sorted by most frequent.
* Escape - Go back to the main group list
* q - quit

If you wish, you can also run in non-interactive mode with `-noninteractive`,
and the list of similar log entries will be printed out along with how many
are in each group and the program will exit. You can also use `-reverse` to
change how the output is sorted.

#### Similarity threshold

The `-threshold` option is used to configure how similar log lines have to be
before they are grouped together. The default is 0.8, meaning 80% of tokens in
a line (tokens are parts of the line separated by whitespace or symbols) need
to be the same for a line to be grouped. If you find that you are seeing many
log lines that are almost identical being printed out, try lowering the
threshold to something like 0.6 instead.

Be careful however, if you set the threshold too low, then lines that are not
really similar at all will be grouped together, and you might miss important
information and end up with groups that look like:

    2015-*-* *:*:* (*) * * * *

### log_multiline

This command combines log entries that span multiple lines into single lines.
By default it will combine any line beginning with whitespace into the
previous line, such as the following postgresql log entry:

    2015-01-01 00:00:00 UTC:1.2.3.4(12345):username@db:[1234]:STATEMENT:
     INSERT INTO some_table(foo, bar)
                VALUES ("FOO", "BAR")

However, you can specify any regular expression to identify lines
that are continuations of previous lines using the `-regex` option.

For example, if every log line is followed by another with the word 'DETAIL:'
in it, and you want to combine them into single entries, you could run:

    log_multiline -regex DETAIL:
