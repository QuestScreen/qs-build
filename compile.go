package main

import (
	"os"
	"os/exec"
	"runtime"
)

func compileQuestscreen() {
	os.Chdir("main")
	var exeName string
	if runtime.GOOS == "windows" {
		exeName = "../questscreen.exe"
	} else {
		exeName = "../questscreen"
	}
	runAndDumpIfVerbose(exec.Command("go", "build", "-o", exeName),
		func(err error, stderr string) {
			logError("failed to compile QuestScreen:")
			logError(err.Error())
			writeErrorLines(stderr)
		})
	os.Chdir("..")
}
