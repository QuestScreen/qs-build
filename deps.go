package main

import (
	"os"
	"os/exec"
)

func ensureAvailable(importPath string) {
	os.Stdout.WriteString("[info] ensuring availability of " + importPath + "\n")
	cmd := exec.Command("go", "get", importPath)
	runAndDumpIfVerbose(cmd, func(err error, stderr string) {
		os.Stdout.WriteString("[error] failed to get " + importPath + ":\n")
		os.Stdout.WriteString("[error] " + err.Error() + "\n")
		writeErrorLines(stderr)
	})
}

func installGo11216() {
	ensureAvailable("golang.org/dl/go1.12.16")
	runAndDumpIfVerbose(exec.Command("go1.12.16", "download"), func(err error, stderr string) {
		os.Stdout.WriteString("[error] failed to get install go1.12.16:\n")
		os.Stdout.WriteString("[error] " + err.Error() + "\n")
		writeErrorLines(stderr)
	})
}

func ensureDepsAvailable() {
	ensureAvailable("github.com/go-bindata/go-bindata/...")
	ensureAvailable("github.com/flyx/askew")
	ensureAvailable("github.com/gopherjs/gopherjs")
	if _, err := exec.LookPath("go1.12.16"); err != nil {
		installGo11216()
	}
}
