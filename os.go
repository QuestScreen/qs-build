package main

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
)

var opts struct {
	Verbose bool `short:"v" long:"verbose" description:"Show verbose debug information"`
}

func runAndCheck(cmd *exec.Cmd, errorHandler func(err error, stderr string)) string {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errorHandler(err, stderr.String())
		os.Exit(1)
	}
	return stdout.String()
}

func writeErrorLines(stderr string) {
	lines := strings.Split(stderr, "\n")
	for _, line := range lines {
		os.Stdout.WriteString("[error] … " + line + "\n")
	}
}

func runAndDumpIfVerbose(cmd *exec.Cmd, errorHandler func(err error, stderr string)) {
	if opts.Verbose {
		os.Stdout.WriteString("[verbose] " + cmd.String() + "\n")
	}
	stdout := strings.TrimSpace(runAndCheck(cmd, errorHandler))
	if opts.Verbose && stdout != "" {
		os.Stdout.WriteString(stdout)
		os.Stdout.WriteString("\n")
	}
}
