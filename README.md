# Log Group Tool

This repository contains a tool that identifies similar lines in a log file and groups them together, replacing any differences with wildcards.

## Install instructions

Install a recent version of go, and run `go build` or `go install`.

## Usage

This command will identify similar log lines and group them together, replacing parts that vary with an asterisk.

This can be very useful in a number of situations:

* Identifying patterns in logs to be able to parse them further
* Identifying log entries that are happening often
* Quickly find exceptional events by looking for log entries that don't occur often, filtering out the noise.

By default, it runs in interactive mode, where you can view the groups and drill down into the individual log lines, or view the different values for each asterisk:

* Up/Down/j/k/PgUp/PgDown/Home/End - Scroll and/or select log entries
* Enter - View all log entries in a given group
* 1-9 - Expand the Nth asterisk and show what values it has in each group,
  sorted by most frequent.
* Escape - Go back to the main group list
* q - quit
* w - Toggle line wrapping

If you wish, you can also run in non-interactive mode with `-noninteractive`,
and the list of similar log entries will be printed out along with how many
are in each group and the program will exit. You can also use `-reverse` to
change how the output is sorted.

### Similarity threshold

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
