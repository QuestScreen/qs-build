package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	flags "github.com/jessevdk/go-flags"
	"golang.org/x/mod/modfile"
)

type command struct {
	cmd, name, description string
	exec                   func()
}

// findQuestScreenModule checks if cwd contains the QuestScreen module.
func findQuestScreenModule() {
	raw, err := ioutil.ReadFile("go.mod")
	if err != nil {
		if !os.IsNotExist(err) {
			os.Stderr.WriteString("[warning] while reading go.mod: ")
			os.Stderr.WriteString(err.Error() + "\n")
		}
	}
	var mod *modfile.File
	if mod, err = modfile.Parse("go.mod", raw, nil); err != nil {
		os.Stderr.WriteString("[warning] unable to read go.mod:\n")
		fmt.Fprintf(os.Stderr, "[warning] %s\n", err.Error())
	} else {
		if mod != nil && mod.Module != nil && mod.Module.Mod.Path == "github.com/QuestScreen/QuestScreen" {
			return
		}
	}
	logError("current directory is not QuestScreen source directory!")
	os.Exit(1)
}

var goPathSrc, goBin string

func main() {
	args, err := flags.Parse(&opts)
	if err != nil {
		logError(err.Error())
		os.Exit(1)
	}
	if opts.Debug {
		if opts.Web != "" && opts.Web != "gopherjs" {
			logError("--debug does not allow web UI backend '%s'\n", opts.Web)
			os.Exit(1)
		}
		opts.Web = "gopherjs"
	}
	if opts.Web == "" {
		opts.Web = "wasm"
	}
	switch opts.Web {
	case "", "wasm":
		opts.wasm = true
	case "gopherjs":
		opts.wasm = false
	default:
		logError("unknown web backend: '%s'", opts.Web)
		os.Exit(1)
	}

	{
		cmd := exec.Command("go", "env", "GOPATH")
		gopath := runAndCheck(cmd, func(err error, stderr string) {
			logError("failed to get GOPATH:")
			logError(err.Error())
			writeErrorLines(stderr)
		})
		goPathFirst := strings.SplitN(gopath, string(os.PathListSeparator), 2)[0]
		goPathSrc = filepath.Join(goPathFirst, "src")
		cmd = exec.Command("go", "env", "GOBIN")
		goBin = runAndCheck(cmd, func(err error, stderr string) {
			logError("failed to get GOBIN:")
			logError(err.Error())
			writeErrorLines(stderr)
		})
		if goBin == "" {
			goBin = filepath.Join(goPathFirst, "bin")
		}
	}

	commands := []command{
		{cmd: "deps", name: "Dependencies",
			description: "Ensures that all dependencies required for building QuestScreen are available",
			exec:        ensureDepsAvailable},
		{cmd: "plugins", name: "Plugins",
			description: "walks the plugins directory and discovers all plugins there. Writes code for loading the plugins in web UI and main app",
			exec:        writePluginLoaders},
		{cmd: "webui", name: "Web UI", description: "compiles web UI to assets/main.js",
			exec: buildWebUI},
		{cmd: "assets", name: "Assets", description: "packages all web files in assets/ into assets/assets.go",
			exec: packAssets},
		{cmd: "compile", name: "Compile", description: "compiles main app", exec: compileQuestscreen},
	}

	commandEnabled := make([]bool, len(commands))
	if len(args) == 0 {
		for i := range commandEnabled {
			commandEnabled[i] = true
		}
	} else {
		foundErrors := false
		for i := range args {
			found := false
			for j := range commands {
				if commands[j].cmd == args[i] {
					found = true
					if commandEnabled[j] {
						logError("duplicate command: '%s'", args[i])
						foundErrors = true
					} else {
						commandEnabled[j] = true
						break
					}
				}
			}
			if !found {
				logError("unknown command: '%s'", args[i])
				foundErrors = true
			}
		}
		if foundErrors {
			os.Exit(1)
		}
	}

	findQuestScreenModule()

	if opts.PluginFile != "" {
		if opts.PluginFile, err = filepath.Abs(opts.PluginFile); err != nil {
			logError(err.Error())
			os.Exit(1)
		}
	}

	if _, err := os.Stat("assets"); err != nil {
		if os.IsNotExist(err) {
			os.Mkdir("assets", 0755)
		} else {
			logError("unable to create directory 'assets':")
			logError(err.Error())
			os.Exit(1)
		}
	}
	for i := range commands {
		if commandEnabled[i] {
			logPhase(commands[i].name)
			commands[i].exec()
		}
	}
}
