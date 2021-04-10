package main

import (
	"os/exec"
)

type ReleaseKind int

const (
	ReleaseSource ReleaseKind = iota
	ReleaseWindowsBinary
)

func release(kind ReleaseKind) {
	mustCond(isWorkingDir(), "cannot release: not in working directory")
	version := writeVersionInfo()
	switch kind {
	case ReleaseSource:
		releaseSource(version)
	case ReleaseWindowsBinary:
		releaseWindowsBinary(version)
	}
}

func releaseSource(version string) {
	archive := exec.Command("git", "archive", "master", "-o", "tmparchive.tar")
	must(archive.Run())
	tar := exec.Command("tar", "--append", "-f", "tmparchive.tar", "versioninfo/versioninfo.go")
	must(tar.Run())

	xz := exec.Command("xz", "-z", "tmparchive.tar")
	must(xz.Run())
	checkRename("tmparchive.tar.xz", "questscreen-"+version+".tar.xz")

	logInfo("created release archive")
}
