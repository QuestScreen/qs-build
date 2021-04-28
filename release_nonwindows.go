//+build !windows

package main

func releaseWindowsBinary(relname string) {
	logError("you must be on Windows to build a Windows binary release")
	finalize(true)
}
