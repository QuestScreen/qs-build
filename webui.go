package main

import (
	"os"
	"os/exec"
)

func buildWebUI() {
	gopherjsRoot := runAndCheck(exec.Command("go1.12.16", "env", "GOROOT"),
		func(err error, stderr string) {
			os.Stdout.WriteString("[error] failed to query go1.12.16 for GOROOT:\n")
			os.Stdout.WriteString("[error] " + err.Error() + "\n")
			writeErrorLines(stderr)
		})

	os.Setenv("GOPHERJS_ROOT", gopherjsRoot)

	os.Chdir("web/main")
	runAndDumpIfVerbose(exec.Command("gopherjs", "build"),
		func(err error, stderr string) {
			os.Stdout.WriteString("[error] failed to compile web UI:\n")
			os.Stdout.WriteString("[error] " + err.Error() + "\n")
			writeErrorLines(stderr)
		})
	os.Chdir("../..")
	os.Rename("web/main/main.js", "assets/main.js")
}
