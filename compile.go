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

func genVersionInfo(out io.Writer) string {
	data := struct {
		Version string
		Date    time.Time
	}{runAndCheck(exec.Command("git", "describe"), nil), time.Now()}

	if err := versioninfoTmpl.Execute(out, data); err != nil {
		logError(err.Error())
	}
	return data.Version
}

func writeVersionInfo() string {
	must(os.MkdirAll("versioninfo", 0755))

	os.Chdir("versioninfo")
	defer os.Chdir("..")

	out, err := os.Create("versioninfo.go")
	must(err)
	defer out.Close()
	return genVersionInfo(out)
}

func isWorkingDir() bool {
	info, err := os.Stat(".git")
	if err != nil {
		if !os.IsNotExist(err) {
			must(err)
		}
		return false
	}
	if info.IsDir() {
		return true
	}
	return false
}

func compileQuestscreen() {
	if isWorkingDir() {
		logInfo("development mode (in git repository)")
		writeVersionInfo()
	} else {
		_, err := os.Stat("versioninfo/versioninfo.go")
		must(err, "cannot compile: not in git repository unable to access versioninfo/versioninfo.go:")
		logInfo("release mode (not in git repository)")
	}

	os.Chdir("main")
	var exeName string
	if runtime.GOOS == "windows" {
		exeName = "../questscreen.exe"
	} else {
		exeName = "../questscreen"
	}
	logInfo("compiling code")
	runAndDumpIfVerbose(exec.Command(goCmd, "build", "-o", exeName),
		func(err error, stderr string) {
			logError("failed to compile QuestScreen:")
			logError(err.Error())
			writeErrorLines(stderr)
		})
	os.Chdir("..")
}
