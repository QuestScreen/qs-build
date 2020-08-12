package main

import (
	"os"
	"os/exec"
)

func compileQuestscreen() {
	os.Chdir("main")
	runAndDumpIfVerbose(exec.Command("go", "build"),
		func(err error, stderr string) {
			os.Stdout.WriteString("[error] failed to compile QuestScreen:\n")
			os.Stdout.WriteString("[error] " + err.Error() + "\n")
			writeErrorLines(stderr)
		})
	os.Chdir("..")
}
