package main

import (
	"os"
	"os/exec"
)

func packAssets() {
	if err := CopyDir("web/assets", "assets"); err != nil {
		logError("failed to copy web/assets folder to assets:")
		logError(err.Error())
		os.Exit(1)
	}
	runAndDumpIfVerbose(exec.Command("go-bindata", "-ignore=.*\\.go", "-o",
		"assets/assets.go", "-pkg", "assets", "-prefix", "assets/", "assets/..."),
		func(err error, stderr string) {
			logError("failed to package assets:")
			logError(err.Error())
			writeErrorLines(stderr)
		})
}
