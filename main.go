package main

import (
	"bufio"
	"fmt"
	"io"
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

var goBin, apiImport, goModPath, goSumPath string
var goModStat, goSumStat os.FileInfo
var goModContent, goSumContent []byte
var externalPlugins bool

// findQuestScreenModule checks if cwd contains the QuestScreen module.
func findQuestScreenModule() {
	var err error
	goModPath, err = filepath.Abs("go.mod")
	must(err, "failed to find go.mod:")
	goModStat, err = os.Stat(goModPath)
	must(err)
	goModContent, err = ioutil.ReadFile(goModPath)
	if err == nil {
		goSumPath, err = filepath.Abs("go.sum")
		must(err)
		goSumStat, err = os.Stat(goSumPath)
		must(err)
		goSumContent, err = ioutil.ReadFile(goSumPath)
		must(err)

		var mod *modfile.File
		if mod, err = modfile.Parse("go.mod", goModContent, nil); err != nil {
			logWarning("unable to parse go.mod: %v", err.Error())
		} else {
			if mod != nil && mod.Module != nil && mod.Module.Mod.Path == "github.com/QuestScreen/QuestScreen" {
				for _, r := range mod.Require {
					if r.Mod.Path == "github.com/QuestScreen/api" {
						apiImport = "github.com/QuestScreen/api"
						if r.Mod.Version != "" {
							apiImport += "@" + r.Mod.Version
						}
						return
					}
				}
				logError("failed to find reference to github.com/QuestScreen/api in go.mod")
				finalize(true)
			}
		}
	} else {
		if !os.IsNotExist(err) {
			logWarning("while reading go.mod: %v", err.Error())
		}
	}

	logError("current directory is not QuestScreen source directory!")
	finalize(true)
}

func finalize(exitError bool) {
	if externalPlugins {
		ioutil.WriteFile(goModPath, goModContent, goModStat.Mode())
		ioutil.WriteFile(goSumPath, goSumContent, goSumStat.Mode())
	}
	if exitError {
		os.Exit(1)
	}
}

func parsePluginsFile(path string) {
	logPhase("Init")

	pFile, err := os.Open(path)
	must(err)
	defer pFile.Close()
	reader := bufio.NewReader(pFile)
	var line string
	lineCount := 0
	for err != io.EOF {
		line, err = reader.ReadString('\n')
		if err != nil && err != io.EOF {
			break
		}
		lineCount++
		if line = strings.TrimSpace(line); line == "" {
			continue
		}
		items := strings.Split(line, " ")
		descr := pluginDescr{line: lineCount}
		if idx := strings.IndexByte(items[len(items)-1], '@'); idx >= 0 {
			descr.version = items[len(items)-1][idx+1:]
			items[len(items)-1] = items[len(items)-1][:idx]
		}
		switch len(items) {
		case 0:
			continue
		case 1:
			descr.importPath = items[0]
			descr.id = filepath.Base(descr.importPath)
		case 2:
			descr.importPath = items[1]
			descr.id = items[0]
		default:
			panic(fmt.Sprintf("%s(%v): too many items in line", opts.PluginFile, lineCount))
		}
		var getPath string
		if descr.version == "" {
			getPath = descr.importPath
		} else {
			getPath = fmt.Sprintf("%s@%s", descr.importPath, descr.version)
		}
		logInfo(fmt.Sprintf("loading plugin '%s' at \"%s\"", descr.id, getPath))
		runAndCheck(exec.Command("go", "get", "-u", getPath), func(err error, stderr string) {
			logError(err.Error())
			writeErrorLines(stderr)
		})
		externalPlugins = true

		descr.dir = runAndCheck(exec.Command("go", "list", "-m", "-f", "{{.Dir}}", descr.importPath), func(err error, stderr string) {
			logError(err.Error())
			writeErrorLines(stderr)
		})

		opts.chosenPlugins = append(opts.chosenPlugins, descr)
	}
}

func main() {
	args, err := flags.Parse(&opts)
	if flags.WroteHelp(err) {
		os.Exit(0)
	}
	must(err)
	if opts.Debug {
		if opts.Web != "" && opts.Web != "gopherjs" {
			logError("--debug does not allow web UI backend '%s'\n", opts.Web)
			finalize(true)
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
		finalize(true)
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
		{cmd: "compile", name: "Compile", description: "compiles main app",
			exec: compileQuestscreen},
	}

	commandEnabled := make([]bool, len(commands))
	doRelease := false
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
				if args[i] == "release" {
					if len(args) != 1 {
						logError("cannot give other commands along with `release`")
						foundErrors = true
					} else {
						doRelease = true
					}
				} else {
					logError("unknown command: '%s'", args[i])
					foundErrors = true
				}
			}
		}
		if foundErrors {
			finalize(true)
		}
	}

	{
		cmd := exec.Command("go", "env", "GOPATH")
		gopath := runAndCheck(cmd, func(err error, stderr string) {
			logError("failed to get GOPATH:")
			logError(err.Error())
			writeErrorLines(stderr)
		})
		goPathFirst := strings.SplitN(gopath, string(os.PathListSeparator), 2)[0]
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

	findQuestScreenModule()

	if doRelease {
		switch opts.Binary {
		case "":
			opts.rKind = ReleaseSource
		case "windows":
			opts.rKind = ReleaseWindowsBinary
		default:
			logError("unknown binary release platform: " + opts.Binary)
			finalize(true)
		}
		release(opts.rKind)
	} else {
		mustCond(opts.Binary == "", "illegal value for --binary: "+opts.Binary,
			"this option may only be given for command 'release'")
	}

	if opts.PluginFile == "" {
		info, err := os.Stat("plugins/plugins.txt")
		if err == nil && !info.IsDir() {
			opts.PluginFile = "plugins/plugins.txt"
		}
	}
	if opts.PluginFile != "" {
		parsePluginsFile(opts.PluginFile)
	}

	if _, err := os.Stat("assets"); err != nil {
		if os.IsNotExist(err) {
			os.Mkdir("assets", 0755)
		} else {
			must(err, "unable to create directory 'assets':")
		}
	}
	for i := range commands {
		if commandEnabled[i] {
			logPhase(commands[i].name)
			commands[i].exec()
		}
	}
	finalize(false)
}
