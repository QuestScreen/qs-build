package main

import (
	"io"
	"os"
	"os/exec"
	"runtime"
	"text/template"
	"time"
)

var versioninfoTmpl = template.Must(template.New("versioninfo").Parse(`
package versioninfo

var CurrentVersion = "{{.Version}}"
var Date = "{{.Date}}"
`))

func genVersionInfo(out io.Writer) {
	data := struct {
		Version string
		Date    time.Time
	}{runAndCheck(exec.Command("git", "describe"), nil), time.Now()}

	if err := versioninfoTmpl.Execute(out, data); err != nil {
		logError(err.Error())
	}
}

func writeVersionInfo() {
	if err := os.MkdirAll("versioninfo", 0755); err != nil {
		logError(err.Error())
		os.Exit(1)
	}

	os.Chdir("versioninfo")
	defer os.Chdir("..")

	out, err := os.Create("versioninfo.go")
	if err != nil {
		logError(err.Error())
		os.Exit(1)
	}
	defer out.Close()
	genVersionInfo(out)
}

func compileQuestscreen() {
	info, err := os.Stat(".git")
	if err != nil {
		if !os.IsNotExist(err) {
			logError(err.Error())
			os.Exit(1)
		}
		logInfo("release mode (not in git repository)")
	} else if info.IsDir() {
		logInfo("development mode (in git repository)")
		writeVersionInfo()
	}

	os.Chdir("main")
	var exeName string
	if runtime.GOOS == "windows" {
		exeName = "../questscreen.exe"
	} else {
		exeName = "../questscreen"
	}
	logInfo("compiling code")
	runAndDumpIfVerbose(exec.Command("go", "build", "-o", exeName),
		func(err error, stderr string) {
			logError("failed to compile QuestScreen:")
			logError(err.Error())
			writeErrorLines(stderr)
		})
	os.Chdir("..")
}
