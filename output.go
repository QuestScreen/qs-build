package main

import (
	"os"

	"github.com/fatih/color"
)

var blueBold = color.New(color.FgBlue).Add(color.Bold)
var bold = color.New(color.FgWhite).Add(color.Bold)
var yellowBold = color.New(color.FgYellow).Add(color.Bold)
var redBold = color.New(color.FgRed).Add(color.Bold)

func logPhase(msg string) {
	blueBold.Print("[phase] ")
	color.Blue(msg)
}

func logInfo(msg string) {
	bold.Print("[info] ")
	os.Stdout.WriteString(msg)
	os.Stdout.WriteString("\n")
}

func logVerbose(msg string) {
	bold.Print("[verbose] ")
	os.Stdout.WriteString(msg)
	os.Stdout.WriteString("\n")
}

func logWarning(msg string, a ...interface{}) {
	yellowBold.Print("[warn] ")
	color.Yellow(msg, a...)
}

func logError(msg string, a ...interface{}) {
	redBold.Print("[error] ")
	color.Red(msg, a...)
}
