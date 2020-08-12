package main

import (
	"os"
	"os/exec"
)

func packAssets() {
	runAndDumpIfVerbose(exec.Command("go-bindata", "-ignore=.*\\.go", "-o", "assets/assets.go", "assets/"),
		func(err error, stderr string) {
			os.Stdout.WriteString("[error] failed to package assets:\n")
			os.Stdout.WriteString("[error] " + err.Error() + "\n")
			writeErrorLines(stderr)
		})
}
