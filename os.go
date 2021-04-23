package main

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
)

type pluginDescr struct {
	id, importPath, version, dir string
	line                         int
}

var opts struct {
	Verbose       bool   `short:"v" long:"verbose" description:"Show verbose debug information"`
	Debug         bool   `short:"d" long:"debug" description:"Build an executable for debugging (includes JS source map and Go sources). Implies --web=gopherjs"`
	Web           string `short:"w" long:"web" description:"Backend to use for the web UI. Either 'wasm' (default) or 'gopherjs'."`
	PluginFile    string `short:"p" long:"pluginFile" description:"Path to a file that contains the import paths of all plugins you want to use" optional:"true"`
	Binary        string `short:"b" long:"binary" description:"use with 'release' to build a binary release. Value specifies platform. Currently, only 'windows' is supported."`
	wasm          bool
	rKind         ReleaseKind
	chosenPlugins []pluginDescr
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
		finalize(true)
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
		finalize(true)
	}
}

func must(err error, errMsg ...string) {
	if err != nil {
		for _, msg := range errMsg {
			logError(msg)
		}
		logError(err.Error())
		finalize(true)
	}
}

func mustCond(cond bool, errMsg ...string) {
	if !cond {
		for _, msg := range errMsg {
			logError(msg)
		}
		finalize(true)
	}
}
