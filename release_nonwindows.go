//+build !windows

package main

func releaseWindowsBinary(version string) {
	logError("you must be on Windows to build a Windows binary release")
	finalize(true)
}
