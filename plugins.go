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
	"runtime"
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

type AssetData struct {
	CSS   []string
	Other []string
}

// PluginDescr loads the content of a questscreen-plugin.yaml file.
type PluginDescr struct {
	Name, importPath, dirPath string
	Modules                   []string
	Templates                 pluginTemplates
	assets                    AssetData
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
	ImportPath, DirPath, ID, Name string
	Modules                       []moduleData
	Templates                     pluginTemplateData
	Assets                        AssetData
}

// Data holds the processed metadata for plugins.
// All maps are transformed into slices for reproducible indexing.
type Data []pluginData

func addAssets(rootPath, dirPath string, assets *AssetData) {
	assetFiles, err := ioutil.ReadDir(dirPath)
	must(err)
	for _, assetFile := range assetFiles {
		filePath := filepath.Join(dirPath, assetFile.Name())
		info, err := os.Stat(filePath)
		must(err)
		if info.IsDir() {
			addAssets(rootPath, filePath, assets)
		} else {
			rel, err := filepath.Rel(rootPath, filePath)
			must(err)
			ext := filepath.Ext(assetFile.Name())
			if strings.ToLower(ext) == ".css" {
				assets.CSS = append(assets.CSS, rel)
			} else {
				assets.Other = append(assets.Other, rel)
			}
		}
	}
}

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
	assetsPath := filepath.Join(path, "web", "assets")
	info, err = os.Stat(assetsPath)
	if err == nil && info.IsDir() {
		addAssets(assetsPath, assetsPath, &p.assets)
	}
	return p, nil
}

func discoverPlugins() map[string]PluginDescr {
	logInfo("discovering plugins")
	plugindirs, err := ioutil.ReadDir(".")
	must(err)
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
			p.dirPath, err = filepath.Abs(plugindir.Name())
			must(err)
			_, ok := plugins[plugindir.Name()]
			mustCond(!ok, "duplicate plugin id: "+plugindir.Name())
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
	must(err)
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
		dirPath := filepath.Join(goPathSrc, importPath)
		p, err := discoverPlugin(importPath, dirPath)
		if err != nil {
			logWarning(err.Error())
			logWarning(importPath + ": skipping")
		} else {
			p.importPath = importPath
			p.dirPath = dirPath
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
			DirPath: value.dirPath,
			Modules: make([]moduleData, len(value.Modules)), Templates: pluginTemplateData{
				Groups:  value.Templates.Groups,
				Scenes:  make([]sceneTmplData, 0, len(value.Templates.Scenes)),
				Systems: make([]systemTmplData, 0, len(value.Templates.Systems)),
			}, Assets: value.assets}
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
	must(err, "unable to transform path to absolute path:")
	logInfo("generating " + abs)

	data, err := os.Stat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return abs
		}
		must(err, "unable to stat file:")
	}
	mustCond(data.Mode().IsRegular(),
		fmt.Sprintf("%v: already exists and not a regular file", abs))
	file, err := os.Open(abs)
	must(err, "unable to open file:")
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
		must(err, "failed to read from existing file:")
	}
	mustCond(generated, abs+":", "… refusing to overwrite file that was not generated by qs-build. delete manually.")
	return abs
}

func writePluginCollector(plugins Data) error {
	path := ensureFileDoesntExistOrIsAutogenerated("plugins.go")
	var writer strings.Builder
	if err := pluginsGoTmpl.Execute(&writer, plugins); err != nil {
		return err
	}
	writeFormatted(writer.String(), path)

	yamlFile, err := os.Create("plugins.yaml")
	if err != nil {
		return err
	}
	defer yamlFile.Close()
	encoder := yaml.NewEncoder(yamlFile)
	defer encoder.Close()
	return encoder.Encode(plugins)
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
	must(webPluginsLoaderTmpl.Execute(&loader, plugins),
		"failed to render plugins.go:")
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
	must(os.Mkdir(dirPath, 0755))
	defer os.RemoveAll(dirPath)

	inspectorExe, err := os.Create(filepath.Join(dirPath, "main.go"))
	must(err, "failed to create file in temporary directory:")
	if err = configLoaderGeneratorTmpl.Execute(inspectorExe, struct {
		Import, ModuleName, ModuleID string
	}{fmt.Sprintf("%v/%v",
		pluginImportPath, moduleName), moduleName, moduleID}); err != nil {
		inspectorExe.Close()
		logError("failed to generate content for config loader generator.")
		panic(err)
	}
	inspectorExe.Close()
	cwd, _ := os.Getwd()
	os.Chdir(dirPath)
	defer os.Chdir(cwd)

	var mainName string
	if runtime.GOOS == "windows" {
		mainName = "main.exe"
	} else {
		mainName = "main"
	}

	runAndDumpIfVerbose(exec.Command("go", "build", "-o", mainName),
		func(err error, stderr string) {
			logError(fmt.Sprintf("%v/%v [tmpdir: %v]:",
				pluginImportPath, moduleName, dirPath))
			logError("failed to build inspector for module configuration:")
			logError(err.Error())
			writeErrorLines(stderr)
		})
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := exec.Command(filepath.Join(dirPath, mainName))
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
				executeInspector(plugin.ImportPath, module.Name,
					module.ImportName, &path)
			} else {
				executeInspector(plugin.ImportPath, module.Name,
					module.ImportName, nil)
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

	must(writePluginCollector(plugins))
	os.Chdir("..")
	writeModuleConfigLoaders(plugins)
	writeWebPluginLoader(plugins)
}
