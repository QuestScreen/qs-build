package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

type sceneTmplRef struct {
	Name     string
	Template string
}

type groupTmpl struct {
	Name        string
	Description string
	Config      map[string]interface{}
	Scenes      []sceneTmplRef
}

type sceneTmpl struct {
	Name        string
	Description string
	Config      map[string]interface{}
}

type systemTmpl struct {
	Name   string
	Config map[string]interface{}
}

type pluginTemplates struct {
	Groups  []groupTmpl
	Scenes  map[string]sceneTmpl
	Systems map[string]systemTmpl
}

// PluginDescr loads the content of a questscreen-plugin.yaml file.
type PluginDescr struct {
	Name, importPath string
	Modules          []string
	Templates        pluginTemplates
}

type sceneTmplData struct {
	ID, Name, Description string
	Config                map[string]interface{}
}

type systemTmplData struct {
	ID, Name string
	Config   map[string]interface{}
}

type pluginTemplateData struct {
	Groups  []groupTmpl
	Scenes  []sceneTmplData
	Systems []systemTmplData
}

type moduleData struct {
	ImportName, Name string
}

type pluginData struct {
	ImportPath, ID, Name string
	Modules              []moduleData
	Templates            pluginTemplateData
}

// Data holds the processed metadata for plugins.
// All maps are transformed into slices for reproducible indexing.
type Data []pluginData

func discoverPlugin(importPath, path string) (PluginDescr, error) {
	yamlPath := filepath.Join(path, "questscreen-plugin.yaml")
	info, err := os.Stat(yamlPath)
	if err != nil {
		return PluginDescr{}, errors.New(importPath + ": " + err.Error())
	}
	if info.IsDir() {
		return PluginDescr{}, errors.New(importPath + ": is a directory")
	}
	description, err := ioutil.ReadFile(yamlPath)
	if err != nil {
		return PluginDescr{}, errors.New(importPath + ": " + err.Error())
	}
	var p PluginDescr
	if err = yaml.Unmarshal(description, &p); err != nil {
		return PluginDescr{}, errors.New(path + ": error while loading YAML: " + err.Error())
	}
	return p, nil
}

func discoverPlugins() map[string]PluginDescr {
	logInfo("discovering plugins")
	plugindirs, err := ioutil.ReadDir(".")
	if err != nil {
		panic(err)
	}
	plugins := make(map[string]PluginDescr)
	for _, plugindir := range plugindirs {
		if !plugindir.IsDir() {
			continue
		}
		p, err := discoverPlugin(plugindir.Name(), plugindir.Name())
		if err != nil {
			logWarning(err.Error())
			logWarning(plugindir.Name() + ": skipping")
		} else {
			p.importPath = "github.com/QuestScreen/QuestScreen/plugins/" + plugindir.Name()
			if _, ok := plugins[plugindir.Name()]; ok {
				panic("duplicate plugin id: " + plugindir.Name())
			}
			plugins[plugindir.Name()] = p
		}
	}
	if opts.PluginFile == "" {
		info, err := os.Stat("plugins.txt")
		if err != nil || info.IsDir() {
			return plugins
		}
		opts.PluginFile = "plugins.txt"
	}
	pFile, err := os.Open(opts.PluginFile)
	if err != nil {
		panic(err)
	}
	defer pFile.Close()
	reader := bufio.NewReader(pFile)
	var line string
	lineCount := 0
	for {
		line, err = reader.ReadString('\n')
		if err != nil && err != io.EOF {
			break
		}
		lineCount++
		line = strings.TrimSpace(line)
		items := strings.Split(line, " ")
		var id, importPath string
		switch len(items) {
		case 0:
			continue
		case 1:
			importPath = items[0]
			id = filepath.Base(importPath)
		case 2:
			importPath = items[1]
			id = items[0]
		default:
			panic(fmt.Sprintf("%s(%v): too many items in line", opts.PluginFile, lineCount))
		}
		p, err := discoverPlugin(importPath, filepath.Join(goPathSrc, importPath))
		if err != nil {
			logWarning(err.Error())
			logWarning(importPath + ": skipping")
		} else {
			p.importPath = importPath
			if _, ok := plugins[id]; ok {
				panic(fmt.Sprintf("%s(%v): duplicate plugin ID `%v`", opts.PluginFile, lineCount, id))
			}
			plugins[id] = p
		}
	}

	return plugins
}

func process(input map[string]PluginDescr) Data {
	ret := make(Data, 0, len(input))
	modCount := 0
	for id, value := range input {
		plugin := pluginData{ImportPath: value.importPath, ID: id, Name: value.Name,
			Modules: make([]moduleData, len(value.Modules)), Templates: pluginTemplateData{
				Groups:  value.Templates.Groups,
				Scenes:  make([]sceneTmplData, 0, len(value.Templates.Scenes)),
				Systems: make([]systemTmplData, 0, len(value.Templates.Systems)),
			}}
		for i, m := range value.Modules {
			plugin.Modules[i] = moduleData{ImportName: fmt.Sprintf("qmod%v", modCount), Name: m}
			modCount++
		}
		for id, s := range value.Templates.Scenes {
			plugin.Templates.Scenes = append(plugin.Templates.Scenes,
				sceneTmplData{
					ID: id, Name: s.Name, Description: s.Description, Config: s.Config})
		}
		for id, s := range value.Templates.Systems {
			plugin.Templates.Systems = append(plugin.Templates.Systems,
				systemTmplData{
					ID: id, Name: s.Name, Config: s.Config})
		}
		ret = append(ret, plugin)
	}
	return ret
}

var pluginsGoTmpl = template.Must(template.New("plugins.go").Funcs(template.FuncMap{
	"yaml": func(value interface{}) (string, error) {
		out, err := yaml.Marshal(value)
		if err != nil {
			return "", err
		}
		return string(out), nil
	},
	"searchPlugin": func(root Data, path string) (int, error) {
		items := strings.Split(path, ".")
		for i, v := range root {
			if v.ID == items[0] {
				return i, nil
			}
		}
		return 0, errors.New("cannot resolve scene template reference, unknown plugin: " + items[0])
	},
	"searchTemplate": func(root Data, path string) (int, error) {
		items := strings.Split(path, ".")
		for _, value := range root {
			if value.ID == items[0] {
				for index, scene := range value.Templates.Scenes {
					if scene.ID == items[1] {
						return index, nil
					}
				}
				return 0, errors.New("cannot resolve scene template reference, unknown scene: " + path)
			}
		}
		return 0, errors.New("cannot resolve scene template reference, unknown plugin: " + path)
	},
}).Parse(`{{$root := . -}}
package plugins

// Code generated by qs-build. DO NOT EDIT.

import (
	"github.com/QuestScreen/api/modules"
	"github.com/QuestScreen/QuestScreen/app"
	{{- range $value := .}}
	{{$value.ID}} "{{$value.ImportPath}}"
	{{- range $value.Modules}}
	{{.ImportName}} "{{$value.ImportPath}}/{{.Name}}"
	{{- end}}
	{{- end}}
)

func LoadPlugins(a app.App) {
	{{range . -}}
	a.AddPlugin("{{.ID}}", &app.Plugin{
		Name: "{{js .Name}}",
		Modules: []*modules.Module{
			{{- range .Modules}}
			&{{.ImportName}}.Descriptor,
			{{- end}}
		},
		SystemTemplates: []app.SystemTemplate{
			{{- range .Templates.Systems}}
			{
				ID: "{{js .ID}}",
				Name: "{{js .Name}}",
				Config: []byte(` + "`" + `{{yaml .Config}}` + "`" + `),
			},
			{{- end}}
		},
		GroupTemplates: []app.GroupTemplate{
			{{- range .Templates.Groups}}
			{
				Name: "{{js .Name}}",
				Description: "{{js .Description}}",
				Config: []byte(` + "`" + `{{yaml .Config}}` + "`" + `),
				Scenes: []app.SceneTmplRef{
					{{- range .Scenes}}
					{
						Name: "{{js .Name}}",
						PluginIndex: {{searchPlugin $root .Template}},
						TmplIndex: {{searchTemplate $root .Template}},
					},
					{{- end}}
				},
			},
			{{- end}}
		},
		SceneTemplates: []app.SceneTemplate{
			{{- range .Templates.Scenes}}
			{
				Name: "{{js .Name}}",
				Description: "{{js .Description}}",
				Config: []byte(` + "`" + `{{yaml .Config}}` + "`" + `),
			},
			{{- end}}
		},
	})
	{{- end}}
}
`))

func ensureFileDoesntExistOrIsAutogenerated(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		logError("unable to transform path to absolute path:")
		logError(err.Error())
		os.Exit(1)
	}
	logInfo("generating " + abs)

	data, err := os.Stat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return abs
		}
		logError("unable to stat file:")
		logError(err.Error())
		os.Exit(1)
	}
	if !data.Mode().IsRegular() {
		logError(fmt.Sprintf("%v: already exists and not a regular file", abs))
		os.Exit(1)
	}
	file, err := os.Open(abs)
	if err != nil {
		logError("unable to open file:")
		logError(err.Error())
		os.Exit(1)
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	generated := false
	for line, err := reader.ReadString('\n'); err == nil; line, err = reader.ReadString('\n') {
		if line == "// Code generated by qs-build. DO NOT EDIT.\n" {
			generated = true
			break
		}
	}

	if err != nil && err != io.EOF {
		logError("failed to read from existing file:")
		logError(err.Error())
		os.Exit(1)
	}
	if !generated {
		logError(abs + ":")
		logError("… refusing to overwrite file that was not generated by qs-build. delete manually.")
		os.Exit(1)
	}
	return abs
}

func writePluginCollector(plugins Data) {
	path := ensureFileDoesntExistOrIsAutogenerated("plugins.go")
	var writer strings.Builder
	pluginsGoTmpl.Execute(&writer, plugins)
	writeFormatted(writer.String(), path)
}

var webPluginsLoaderTmpl = template.Must(template.New("webPluginsLoader").Funcs(
	template.FuncMap{
		"inc": func(i int) int {
			return i + 1
		},
	}).Parse(
	`package main

// Code generated by qs-build. DO NOT EDIT.

import (
	"github.com/QuestScreen/QuestScreen/web"
	{{- range $p := .}}
	{{- range $p.Modules}}
	{{.ImportName}} "{{$p.ImportPath}}/web/{{.Name}}"
	{{- end}}
	{{- end}}
)

func registerPlugins(loader *pluginLoader) error {
	{{- range $index, $value := .}}
	loader.id = "{{$value.ID}}"
	loader.index = {{$index}}
	{{- range $value.Modules}}
	if err := loader.RegisterModule("{{.Name}}", {{.ImportName}}.NewState, web.ConfigDescriptor{{.ImportName}});
		err != nil {
		return err
	}
	{{- end}}
	{{- end}}
	return nil
}
`))

func writeWebPluginLoader(plugins Data) {
	path := ensureFileDoesntExistOrIsAutogenerated("web/main/plugins.go")
	var loader strings.Builder
	if err := webPluginsLoaderTmpl.Execute(&loader, plugins); err != nil {
		logError("failed to render plugins.go:")
		logError(err.Error())
		os.Exit(1)
	}
	writeFormatted(loader.String(), path)
}

var configLoaderGeneratorTmpl = template.Must(template.New("configLoaderGenerator").Parse(
	`package main

// Code generated by qs-build. DO NOT EDIT.

import (
	"{{.Import}}"
	"github.com/QuestScreen/qs-build/inspector"
)

func main() {
	err := inspector.InspectConfig("{{.ModuleName}}", "{{.ModuleID}}", {{.ModuleName}}.Descriptor.DefaultConfig)
	if err != nil {
		panic(err)
	}
}

`))

// if path is nil, the inspector cannot be created inside the plugin because it
// is part of a different module.
func executeInspector(pluginImportPath, moduleName, moduleID string,
	path *string) {
	targetPath := ensureFileDoesntExistOrIsAutogenerated(
		filepath.Join("web", "configitems"+moduleID+".go"))

	if path == nil {
		panic("external-module plugins not implemented")
	}
	var dirPath string
	for i := 0; ; i++ {
		if i == 0 {
			dirPath = filepath.Join(*path, "tmp")
		} else {
			dirPath = filepath.Join(*path, fmt.Sprintf("tmp%v", i))
		}
		if _, err := os.Stat(dirPath); err != nil {
			if os.IsNotExist(err) {
				break
			}
			panic(err)
		}
	}
	if err := os.Mkdir(dirPath, 0755); err != nil {
		panic(err)
	}
	defer os.RemoveAll(dirPath)

	inspectorExe, err := os.Create(filepath.Join(dirPath, "main.go"))
	if err != nil {
		logError("failed to create file in temporary directory:")
		logError(err.Error())
		os.Exit(1)
	}
	if err = configLoaderGeneratorTmpl.Execute(inspectorExe, struct {
		Import, ModuleName, ModuleID string
	}{fmt.Sprintf("%v/%v",
		pluginImportPath, moduleName), moduleName, moduleID}); err != nil {
		inspectorExe.Close()
		logError("failed to generate content for config loader generator:")
		logError(err.Error())
		os.Exit(1)
	}
	inspectorExe.Close()
	cwd, _ := os.Getwd()
	os.Chdir(dirPath)
	defer os.Chdir(cwd)

	buildCmd := exec.Command("go", "build", "-o", "main")
	buildCmd.Env = os.Environ()
	buildCmd.Env = append(buildCmd.Env, "GO111MODULE=off")

	runAndDumpIfVerbose(buildCmd,
		func(err error, stderr string) {
			logError(fmt.Sprintf("%v/%v [tmpdir: %v]:",
				pluginImportPath, moduleName, dirPath))
			logError("failed to build inspector for module configuration:")
			logError(err.Error())
			writeErrorLines(stderr)
		})
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := exec.Command(filepath.Join(dirPath, "main"))
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err = cmd.Run(); err != nil {
		logError(fmt.Sprintf("%v/%v [tmpdir: %v]:",
			pluginImportPath, moduleName, dirPath))
		logError("failed to execute inspector for module configuration:")
		logError(err.Error())
		writeErrorLines(stderr.String())
		os.Exit(1)
	}

	writeFormatted(stdout.String(), targetPath)
}

func writeModuleConfigLoaders(plugins Data) {
	for _, plugin := range plugins {
		for _, module := range plugin.Modules {
			if strings.HasPrefix(plugin.ImportPath, "github.com/QuestScreen/QuestScreen/") {
				path, err := filepath.Abs(
					plugin.ImportPath[len("github.com/QuestScreen/QuestScreen/"):])
				if err != nil {
					panic(err)
				}
				executeInspector(plugin.ImportPath, module.Name, module.ImportName,
					&path)
			} else {
				executeInspector(plugin.ImportPath, module.Name, module.ImportName, nil)
			}
		}
	}
}

func writePluginLoaders() {
	os.Chdir("plugins")
	plugins := process(discoverPlugins())

	for _, plugin := range plugins {
		for _, value := range plugin.Templates.Systems {
			value.Config["name"] = value.Name
		}
	}

	writePluginCollector(plugins)
	os.Chdir("..")
	writeModuleConfigLoaders(plugins)
	writeWebPluginLoader(plugins)
}
