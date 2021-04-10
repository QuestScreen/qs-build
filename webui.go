package main

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func copy(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}
	return out.Close()
}

func buildWebUI() {
	logInfo("running askew")
	var cmd *exec.Cmd
	if opts.wasm {
		cmd = exec.Command("askew", "-o", "assets", "-b", "wasm", "-d", "plugins/plugins.yaml",
			"--exclude", "app,assets,build-doc,data,display,main,shared", ".")
	} else {
		cmd = exec.Command("askew", "-o", "assets", "-b", "gopherjs", "-d", "plugins/plugins.yaml",
			"--exclude", "app,assets,build-doc,data,display,main,shared", ".")
	}
	runAndDumpIfVerbose(cmd,
		func(err error, stderr string) {
			logError("failed to run askew:")
			logError(err.Error())
			writeErrorLines(stderr)
		})
	os.Chdir("web/main")

	if opts.wasm {
		logInfo("compiling code to WASM")
		cmd := exec.Command("go", "build", "-o", "main.wasm")
		cmd.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")
		runAndDumpIfVerbose(cmd, func(err error, stderr string) {
			logError("failed to compile web UI:")
			logError(err.Error())
			writeErrorLines(stderr)
		})
		goroot := runAndCheck(exec.Command("go", "env", "GOROOT"), func(err error, stderr string) {
			logError("while trying to get GOROOT:")
			logError(err.Error())
			writeErrorLines(stderr)
			os.Exit(1)
		})
		os.Chdir("../..")
		checkRename("web/main/main.wasm", "assets/main.wasm")
		if err := copy(filepath.Join(goroot, "misc/wasm/wasm_exec.js"), "assets/wasm_exec.js"); err != nil {
			logError("while copying 'wasm_exec.js:")
			logError(err.Error())
			os.Exit(1)
		}
	} else {
		logInfo("compiling code to JavaScript")
		gopherjsRoot := runAndCheck(exec.Command("go1.12.16", "env", "GOROOT"),
			func(err error, stderr string) {
				logError("failed to query go1.12.16 for GOROOT:")
				logError(err.Error())
				writeErrorLines(stderr)
			})

		cmd := exec.Command("gopherjs", "build")
		if runtime.GOOS == "windows" {
			cmd.Env = append(os.Environ(), "GOPHERJS_GOROOT="+gopherjsRoot, "GOOS=linux")
		} else {
			cmd.Env = append(os.Environ(), "GOPHERJS_GOROOT="+gopherjsRoot)
		}
		runAndDumpIfVerbose(cmd,
			func(err error, stderr string) {
				logError("failed to compile web UI:")
				logError(err.Error())
				writeErrorLines(stderr)
			})

		os.Chdir("../..")
		checkRename("web/main/main.js", "assets/main.js")
		checkRename("web/main/main.js.map", "assets/main.js.map")
	}
}
