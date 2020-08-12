package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/jessevdk/go-flags"
	"golang.org/x/mod/modfile"
)

type command struct {
	cmd, name, description string
	exec                   func()
}

// isInLocalQuestScreenRepo checks if cwd is the root directory of the
// QuestScreen source by trying to read the local go.mod file.
func isInLocalQuestScreenRepo() bool {
	raw, err := ioutil.ReadFile("go.mod")
	if err != nil {
		if !os.IsNotExist(err) {
			os.Stderr.WriteString("[warning] while reading go.mod: ")
			os.Stderr.WriteString(err.Error() + "\n")
			os.Stderr.WriteString("[warning] assuming we're not in a local QuestScreen source root\n")
		}
		return false
	}
	var mod *modfile.File
	if mod, err = modfile.Parse("go.mod", raw, nil); err != nil {
		os.Stderr.WriteString("[warning] unable to read go.mod:\n")
		fmt.Fprintf(os.Stderr, "[warning] %s\n", err.Error())
		os.Stderr.WriteString("[warning] assuming we're not in a local QuestScreen source root\n")
		return false
	}
	return mod.Module.Mod.Path == "github.com/QuestScreen/QuestScreen"
}

func main() {
	args, err := flags.Parse(&opts)
	if err != nil {
		panic(err)
	}

	commands := []command{
		command{cmd: "deps", name: "Dependencies",
			description: "Ensures that all dependencies required for building QuestScreen are available",
			exec:        ensureDepsAvailable},
		command{cmd: "plugins", name: "Plugins",
			description: "walks the plugins directory and discovers all plugins there. Writes code for loading the plugins in web UI and main app",
			exec:        writePluginLoaders},
		command{cmd: "webui", name: "Web UI", description: "compiles web UI to assets/main.js",
			exec: buildWebUI},
		command{cmd: "assets", name: "Assets", description: "packages all web files in assets/ into assets/assets.go",
			exec: packAssets},
		command{cmd: "compile", name: "Compile", description: "compiles main app", exec: compileQuestscreen},
	}

	commandEnabled := make([]bool, len(commands))
	if len(args) == 0 {
		for i := range commandEnabled {
			commandEnabled[i] = true
		}
	} else {
		for i := range args {
			found := false
			for j := range commands {
				if commands[j].cmd == args[i] {
					found = true
					if commandEnabled[j] {
						panic("duplicate command: " + args[i])
					}
					commandEnabled[j] = true
					break
				}
			}
			if !found {
				panic("unknown command: " + args[i])
			}
		}
	}

	local := isInLocalQuestScreenRepo()
	if local {
		os.Stdout.WriteString("[init] using cwd as QuestScreen source directory\n")
	} else {
		os.Stdout.WriteString("[init] not in QuestScreen source root; downloading it\n")
		panic("out-of-source build not implemented")
	}

	for i := range commands {
		if commandEnabled[i] {
			os.Stdout.WriteString("[phase] " + commands[i].name + "\n")
			commands[i].exec()
		}
	}
}
