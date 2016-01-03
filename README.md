# Logtools

This repository contains a few simple command line tools for working with log
files. The tools are:

* log_group - identifies similar lines in a log file and groups them
  together, replacing any differences with wildcards.

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
