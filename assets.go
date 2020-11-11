package main

import (
	"os/exec"
)

func packAssets() {
	runAndDumpIfVerbose(exec.Command("go-bindata", "-ignore=.*\\.go", "-o", "assets/assets.go", "assets/"),
		func(err error, stderr string) {
			logError("failed to package assets:")
			logError(err.Error())
			writeErrorLines(stderr)
		})
}
