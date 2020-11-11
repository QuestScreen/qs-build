package main

import (
	"os"
	"os/exec"
)

func compileQuestscreen() {
	os.Chdir("main")
	runAndDumpIfVerbose(exec.Command("go", "build"),
		func(err error, stderr string) {
			logError("failed to compile QuestScreen:")
			logError(err.Error())
			writeErrorLines(stderr)
		})
	os.Chdir("..")
}
