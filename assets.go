package main

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

func packAssets() {
	required := make(map[string]struct{})
	required["index.html"] = struct{}{}
	if opts.wasm {
		required["main.wasm"] = struct{}{}
		required["wasm_exec.js"] = struct{}{}
	} else {
		required["main.js"] = struct{}{}
		required["main.js.map"] = struct{}{}
	}

	if _, err := os.Stat("assets"); err != nil {
		if os.IsNotExist(err) {
			logError("`assets` directory not existing")
			logError("please execute command `webui` before `assets`")
			finalize(true)
		}
		logError("failed to query for assets directory:")
		logError(err.Error())
		finalize(true)
	} else {
		logInfo("cleaning up")
		files, err := ioutil.ReadDir("assets")
		if err != nil {
			logError("failed to read `assets` directory:")
			logError(err.Error())
			finalize(true)
		}
		for _, file := range files {
			if _, ok := required[file.Name()]; ok {
				delete(required, file.Name())
			} else {
				if err = os.RemoveAll(filepath.Join("assets", file.Name())); err != nil {
					logError("failed to remove assets/" + file.Name() + ":")
					logError(err.Error())
					finalize(true)
				}
			}
		}
		for key := range required {
			logError("The file assets/%s is missing", key)
			logError("Please run command `webui` before `assets`")
			finalize(true)
		}
	}

	apiPath := runAndCheck(exec.Command(goCmd, "list", "-f", "{{.Dir}}", "-m",
		"github.com/QuestScreen/api"), func(err error, stderr string) {
		logError("failed to get path to api resources:")
		logError(err.Error())
		writeErrorLines(stderr)
	})
	logInfo("copying assets from api resources")
	if err := CopyDir(filepath.Join(apiPath, "web", "assets"), "assets"); err != nil {
		logError("failed to copy api resources:")
		logError(err.Error())
		finalize(true)
	}

	logInfo("copying web assets into `assets`")
	if err := CopyDir("web/assets", "assets"); err != nil {
		logError("failed to copy `web/assets` folder to `assets`:")
		logError(err.Error())
		finalize(true)
	}

	{
		var plugins Data
		pluginYamlFile, err := ioutil.ReadFile(filepath.Join("plugins", "plugins.yaml"))
		mustCond(err == nil, "missing file: plugins/plugins.yaml", "please run command `plugins` before `assets`.")
		must(yaml.Unmarshal(pluginYamlFile, &plugins), "failed to read plugins/plugins.yaml:")
		for _, p := range plugins {
			logInfo("copying assets of plugin " + p.ID)
			pluginAssetsPath := filepath.Join("assets", p.ID)
			if err = os.Mkdir(pluginAssetsPath, 0755); err != nil {
				logError(err.Error())
				finalize(true)
			}
			for _, a := range append(p.Assets.CSS, p.Assets.Other...) {
				assetTargetPath := filepath.Join(pluginAssetsPath, a)
				os.MkdirAll(filepath.Dir(assetTargetPath), 0755)
				must(CopyFile(filepath.Join(p.DirPath, "web", "assets", a), assetTargetPath))
			}
		}
	}

	logInfo("packaging assets into assets/assets.go")
	goBindataCmd := filepath.Join(goBin, "go-bindata")
	runAndDumpIfVerbose(exec.Command(goBindataCmd, "-ignore=assets\\.go",
		"-ignore=main\\.js\\.map", "-o", "assets/assets.go", "-pkg", "assets",
		"-prefix", "assets/", "assets/..."),
		func(err error, stderr string) {
			logError("failed to package assets:")
			logError(err.Error())
			writeErrorLines(stderr)
		})

	if opts.Debug {
		logInfo("bunding Go source files for JavaScript debugging")
		runAndCheck(exec.Command(goCmd, "mod", "vendor"),
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
			finalize(true)
		}
		for _, item := range items {
			if item.IsDir() {
				if err = os.Rename(filepath.Join("vendor", item.Name()),
					filepath.Join("assets", item.Name())); err != nil {
					logError("failed to rename `vendor/" + item.Name() + "` to assets/" +
						item.Name() + ":")
					logError(err.Error())
					logError("after solving the problem, remove `vendor` before trying again")
					finalize(true)
				}
			}
		}
		if err = os.RemoveAll("vendor"); err != nil {
			logError("failed to remove `vendor` directory:")
			logError(err.Error())
			logError("after solving the problem, remove `vendor` before trying again")
			finalize(true)
		}
		must(os.MkdirAll("assets/github.com/QuestScreen/QuestScreen", 0755),
			"failed to create directory assets/github.com/QuestScreen/QuestScreen:")
		must(CopyDir("web", "assets/github.com/QuestScreen/QuestScreen/web"),
			"failed to copy Go sources into assets:")
		os.RemoveAll("assets/github.com/QuestScreen/QuestScreen/web/assets")
		logInfo("re-packaging to include source files")
		runAndDumpIfVerbose(exec.Command(goBindataCmd, "-ignore=assets\\.go",
			"-o", "assets/assets.go", "-pkg", "assets",
			"-prefix", "assets/", "assets/..."),
			func(err error, stderr string) {
				logError("failed to package assets:")
				logError(err.Error())
				writeErrorLines(stderr)
			})
	}
}
