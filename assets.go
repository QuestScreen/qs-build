package main

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
)

func packAssets() {
	if _, err := os.Stat("assets"); err != nil {
		if os.IsNotExist(err) {
			logError("`assets` directory not existing")
			logError("please execute command `webui` before `assets`")
			os.Exit(1)
		}
		logError("failed to query for assets directory:")
		logError(err.Error())
		os.Exit(1)
	} else {
		logInfo("cleaning up")
		files, err := ioutil.ReadDir("assets")
		if err != nil {
			logError("failed to read `assets` directory:")
			logError(err.Error())
			os.Exit(1)
		}
		found := 0
		for _, file := range files {
			if file.Name() != "main.js" && file.Name() != "main.js.map" &&
				file.Name() != "index.html" {
				logInfo("removing " + filepath.Join("assets", file.Name()))
				if err = os.RemoveAll(filepath.Join("assets", file.Name())); err != nil {
					logError("failed to remove assets/" + file.Name() + ":")
					logError(err.Error())
					os.Exit(1)
				}
			} else {
				found++
			}
		}
		if found < 3 {
			logError("one of these files in assets is missing: main.js main.js.map index.html")
			logError("please run command `webui` before `assets`")
			os.Exit(1)
		}
	}

	logInfo("copying web assets into `assets`")
	if err := CopyDir("web/assets", "assets"); err != nil {
		logError("failed to copy `web/assets` folder to `assets`:")
		logError(err.Error())
		os.Exit(1)
	}
	// TODO: assets from plugins
	logInfo("packaging assets into assets/assets.go")
	runAndDumpIfVerbose(exec.Command("go-bindata", "-ignore=assets\\.go",
		"-ignore=main\\.js\\.map", "-o", "assets/assets.go", "-pkg", "assets",
		"-prefix", "assets/", "assets/..."),
		func(err error, stderr string) {
			logError("failed to package assets:")
			logError(err.Error())
			writeErrorLines(stderr)
		})

	if opts.Debug {
		logInfo("bunding Go source files for JavaScript debugging")
		runAndCheck(exec.Command("go", "mod", "vendor"),
			func(err error, stderr string) {
				logError("failed to execute `go mod vendor`:")
				logError(err.Error())
				writeErrorLines(stderr)
			})
		items, err := ioutil.ReadDir("vendor")
		if err != nil {
			logError("failed to read generated `vendor` directory:")
			logError(err.Error())
			logError("after solving the problem, remove `vendor` before trying again")
			os.Exit(1)
		}
		for _, item := range items {
			if item.IsDir() {
				if err = os.Rename(filepath.Join("vendor", item.Name()),
					filepath.Join("assets", item.Name())); err != nil {
					logError("failed to rename `vendor/" + item.Name() + "` to assets/" +
						item.Name() + ":")
					logError(err.Error())
					logError("after solving the problem, remove `vendor` before trying again")
					os.Exit(1)
				}
			}
		}
		if err = os.RemoveAll("vendor"); err != nil {
			logError("failed to remove `vendor` directory:")
			logError(err.Error())
			logError("after solving the problem, remove `vendor` before trying again")
			os.Exit(1)
		}
		if err := os.MkdirAll("assets/github.com/QuestScreen/QuestScreen", 0755); err != nil {
			logError("failed to create directory assets/github.com/QuestScreen/QuestScreen:")
			logError(err.Error())
			os.Exit(1)
		}
		if err := CopyDir("web", "assets/github.com/QuestScreen/QuestScreen/web"); err != nil {
			logError("failed to copy Go sources into assets:")
			logError(err.Error())
			os.Exit(1)
		}
		os.RemoveAll("assets/github.com/QuestScreen/QuestScreen/web/assets")
		logInfo("re-packaging to include source files")
		runAndDumpIfVerbose(exec.Command("go-bindata", "-ignore=assets\\.go",
			"-o", "assets/assets.go", "-pkg", "assets",
			"-prefix", "assets/", "assets/..."),
			func(err error, stderr string) {
				logError("failed to package assets:")
				logError(err.Error())
				writeErrorLines(stderr)
			})
	}
}
