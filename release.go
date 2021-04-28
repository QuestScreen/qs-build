package main

import (
	"os/exec"
	"strings"
)

type ReleaseKind int

const (
	ReleaseSource ReleaseKind = iota
	ReleaseWindowsBinary
)

func release(kind ReleaseKind) {
	mustCond(isWorkingDir(), "cannot release: not in working directory")

	res := runAndCheck(exec.Command("git", "status", "--porcelain"),
		func(err error, stderr string) {
			logError("failed to check working directory git status:")
			logError(err.Error())
		})
	if strings.TrimSpace(res) != "" {
		logError("working directory not clean!")
		logError("please commit uncommited changes and remove untracked files before building a release.")
		finalize(true)
	}

	relname := "questscreen-" + writeVersionInfo()

	switch kind {
	case ReleaseSource:
		releaseSource(relname)
	case ReleaseWindowsBinary:
		releaseWindowsBinary(relname)
	}
}

func releaseSource(relname string) {

	archive := exec.Command("git", "archive", "master", "-o", "tmparchive.tar")
	must(archive.Run())
	tar := exec.Command("tar", "--append", "-f", "tmparchive.tar", "versioninfo/versioninfo.go")
	must(tar.Run())

	xz := exec.Command("xz", "-z", "tmparchive.tar")
	must(xz.Run())
	checkRename("tmparchive.tar.xz", relname+".tar.xz")

	logInfo("created release archive")
}
