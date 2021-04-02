package main

import (
	"bytes"
	"os"
	"os/exec"
)

func ensureAvailable(importPath string) {
	logInfo("ensuring availability of " + importPath)
	cmd := exec.Command("go", "get", importPath)
	runAndDumpIfVerbose(cmd, func(err error, stderr string) {
		logError("failed to get " + importPath + ":")
		logError(err.Error())
		writeErrorLines(stderr)
	})
}

func downloadGo11216() {
	os.Stdout.WriteString("[info] downloading go1.12.16 SDK\n")
	runAndDumpIfVerbose(exec.Command("go1.12.16", "download"), func(err error, stderr string) {
		logError("failed to download go1.12.16 SDK:")
		logError(err.Error())
		writeErrorLines(stderr)
	})
}

func installGo11216() {
	ensureAvailable("golang.org/dl/go1.12.16")
	downloadGo11216()
}

func ensureDepsAvailable() {
	ensureAvailable("golang.org/x/tools/cmd/goimports")
	ensureAvailable("github.com/go-bindata/go-bindata/...")
	ensureAvailable("github.com/flyx/askew")
	if !opts.wasm {
		ensureAvailable("github.com/gopherjs/gopherjs")
		if _, err := exec.LookPath("go1.12.16"); err != nil {
			installGo11216()
		} else {
			// could be that the command is available but the SDK is not downloaded
			cmd := exec.Command("go1.12.16", "version")
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr
			if err := cmd.Run(); err != nil {
				downloadGo11216()
			}
		}
	}
}
