package main

import (
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
)

func writeFormatted(goCode string, file string) {
	fmtcmd := exec.Command("goimports")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	fmtcmd.Stdout = &stdout
	fmtcmd.Stderr = &stderr

	stdin, err := fmtcmd.StdinPipe()
	if err != nil {
		panic("unable to create stdin pipe: " + err.Error())
	}
	io.WriteString(stdin, goCode)
	stdin.Close()

	if err := fmtcmd.Run(); err != nil {
		logError("failed to format Go code:")
		logError(err.Error())
		logError("stderr output:")
		writeErrorLines(stderr.String())
		logError("input:")
		log.Println(goCode)
		os.Exit(1)
	}

	must(ioutil.WriteFile(file, stdout.Bytes(), os.ModePerm))
}
