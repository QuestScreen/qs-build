package main

import (
	"os"
	"os/exec"
	"strings"
)

func buildWebUI() {
	logInfo("running askew")
	runAndDumpIfVerbose(exec.Command("askew", "-o", "assets", "--exclude",
		"app,assets,build-doc,data,display,main,shared", "."),
		func(err error, stderr string) {
			logError("failed to run askew:")
			logError(err.Error())
			writeErrorLines(stderr)
		})

	logInfo("compiling code to JavaScript")
	gopherjsRoot := runAndCheck(exec.Command("go1.12.16", "env", "GOROOT"),
		func(err error, stderr string) {
			logError("failed to query go1.12.16 for GOROOT:")
			logError(err.Error())
			writeErrorLines(stderr)
		})

	os.Chdir("web/main")
	cmd := exec.Command("gopherjs", "build")
	cmd.Env = append(os.Environ(), "GOPHERJS_GOROOT="+strings.TrimSpace(gopherjsRoot))
	runAndDumpIfVerbose(cmd,
		func(err error, stderr string) {
			logError("failed to compile web UI:")
			logError(err.Error())
			writeErrorLines(stderr)
		})

	os.Chdir("../..")
	os.Rename("web/main/main.js", "assets/main.js")
	os.Rename("web/main/main.js.map", "assets/main.js.map")
}
