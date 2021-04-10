//+build !windows

package main

import "os"

func releaseWindowsBinary(version string) {
	logError("you must be on Windows to build a Windows binary release")
	os.Exit(1)
}
