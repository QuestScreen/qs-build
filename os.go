package main

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
)

var opts struct {
	Verbose    bool   `short:"v" long:"verbose" description:"Show verbose debug information"`
	Debug      bool   `short:"d" long:"debug" description:"Build an executable for debugging (includes JS source map and Go sources). Implies --web=gopherjs"`
	Web        string `short:"w" long:"web" description:"Backend to use for the web UI. Either 'wasm' (default) or 'gopherjs'."`
	PluginFile string `shord:"p" long:"pluginFile" description:"Path to a file that contains the import paths of all plugins you want to use" optional:"true"`
	wasm       bool
}

func runAndCheck(cmd *exec.Cmd, errorHandler func(err error, stderr string)) string {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errorHandler(err, stderr.String())
		output := strings.TrimSpace(stdout.String())
		if len(output) > 0 {
			logError("output:")
			os.Stdout.WriteString(stdout.String())
		}
		os.Exit(1)
	}
	return strings.TrimSpace(stdout.String())
}

func writeErrorLines(stderr string) {
	lines := strings.Split(stderr, "\n")
	for len(lines) > 0 && len(lines[len(lines)-1]) == 0 {
		lines = lines[:len(lines)-1]
	}
	for _, line := range lines {
		logError("… " + line)
	}
}

func runAndDumpIfVerbose(cmd *exec.Cmd, errorHandler func(err error, stderr string)) {
	if opts.Verbose {
		logVerbose(cmd.String())
	}
	stdout := runAndCheck(cmd, errorHandler)
	if opts.Verbose && stdout != "" {
		os.Stdout.WriteString(stdout)
		os.Stdout.WriteString("\n")
	}
}

func checkRename(src, dst string) {
	if err := os.Rename(src, dst); err != nil {
		logError("while renaming '%s' to '%s':", src, dst)
		logError(err.Error())
		os.Exit(1)
	}
}
