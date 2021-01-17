package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
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
	Name      string
	Modules   []string
	Templates pluginTemplates
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

type pluginData struct {
	ID, Name  string
	Modules   []string
	Templates pluginTemplateData
}

// Data holds the processed metadata for plugins.
// All maps are transformed into slices for reproducible indexing.
type Data []pluginData

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
		path := filepath.Join(plugindir.Name(), "questscreen-plugin.yaml")
		info, err := os.Stat(path)
		if err != nil {
			logWarning(path + ": " + err.Error())
			logWarning(plugindir.Name() + ": skipping")
			continue
		}
		if info.IsDir() {
			logWarning(path + ": is a directory")
			logWarning(plugindir.Name() + ": skipping")
			continue
		}
		description, err := ioutil.ReadFile(path)
		if err != nil {
			logWarning(path + ": " + err.Error())
			logWarning(plugindir.Name() + ": skipping")
			continue
		}
		var p PluginDescr
		if err = yaml.Unmarshal(description, &p); err != nil {
			logWarning(path + ": error while loading YAML: " + err.Error())
			logWarning(plugindir.Name() + ": skipping")
			continue
		}
		plugins[plugindir.Name()] = p
	}
	return plugins
}

func process(input map[string]PluginDescr) Data {
	ret := make(Data, 0, len(input))
	for id, value := range input {
		plugin := pluginData{ID: id, Name: value.Name, Modules: value.Modules,
			Templates: pluginTemplateData{
				Groups:  value.Templates.Groups,
				Scenes:  make([]sceneTmplData, 0, len(value.Templates.Scenes)),
				Systems: make([]systemTmplData, 0, len(value.Templates.Systems)),
			}}
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
// generated by qs-build. do not edit.

package plugins

import (
	"github.com/QuestScreen/api/modules"
	"github.com/QuestScreen/QuestScreen/app"
	{{- range $value := .}}
	"github.com/QuestScreen/QuestScreen/plugins/{{$value.ID}}"
	{{- range $value.Modules}}
	"github.com/QuestScreen/QuestScreen/plugins/{{$value.ID}}/{{.}}"
	{{- end}}
	{{- end}}
)

func LoadPlugins(a app.App) {
	{{range . -}}
	a.AddPlugin("{{.ID}}", &app.Plugin{
		Name: "{{js .Name}}",
		Modules: []*modules.Module{
			{{- range .Modules}}
			&{{.}}.Descriptor,
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

func ensureFileDoesntExistOrIsAutogenerated(path string) {
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
			return
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
	line, isPrefix, err := reader.ReadLine()
	if err != nil {
		logError("failed to read from existing file:")
		logError(err.Error())
		os.Exit(1)
	}
	if isPrefix {
		return
	}
	if string(line) != "// generated by qs-build. do not edit." {
		logError(abs + ":")
		logError("… refusing to overwrite file that was not generated by qs-build. delete manually.")
		os.Exit(1)
	}
}

func writePluginCollector(plugins Data) {
	ensureFileDoesntExistOrIsAutogenerated("plugins.go")
	var writer strings.Builder
	pluginsGoTmpl.Execute(&writer, plugins)
	writeFormatted(writer.String(), "plugins.go")
}

var webPluginLoaderTmpl = template.Must(template.New("webPluginLoader").Parse(
	`// generated by qs-build. do not edit.
package web

{{$pluginName := .Name}}
import (
	{{- range .Modules}}
	{{.}} "github.com/QuestScreen/QuestScreen/plugins/{{$pluginName}}/web/{{.}}"
	{{- end}}
	"github.com/QuestScreen/api/web"
)

func RegisterModules(reg web.PluginRegistrator) error {
	{{- range .Modules}}
	if err := reg.RegisterModule("{{.}}", {{.}}.NewState); err != nil {
		return err
	}
	{{- end}}
	return nil
}
`))

func writeModuleLoader(pluginName string, modules []string) {
	ensureFileDoesntExistOrIsAutogenerated("web/init.go")
	var writer strings.Builder
	data := struct {
		Name    string
		Modules []string
	}{Name: pluginName, Modules: modules}
	webPluginLoaderTmpl.Execute(&writer, data)
	writeFormatted(writer.String(), "web/init.go")
}

var webPluginsLoaderTmpl = template.Must(template.New("webPluginsLoader").Funcs(
	template.FuncMap{
		"inc": func(i int) int {
			return i + 1
		},
	}).Parse(
	`// generated by qs-build. do not edit.

package main

import (
	{{- range .}}
	{{.ID}} "github.com/QuestScreen/QuestScreen/plugins/{{.ID}}/web"
	{{- end}}
)

func registerPlugins(loader *pluginLoader) error {
	{{- range $index, $value := .}}
	loader.id = "{{$value.ID}}"
	loader.index = {{$index}}
  if err := {{$value.ID}}.RegisterModules(loader); err != nil {
		return err
	}
	{{- end}}
	return nil
}
`))

func writeWebPluginLoader(plugins Data) {
	ensureFileDoesntExistOrIsAutogenerated("web/main/plugins.go")
	var loader strings.Builder
	if err := webPluginsLoaderTmpl.Execute(&loader, plugins); err != nil {
		logError("failed to render plugins.go:")
		logError(err.Error())
		os.Exit(1)
	}
	writeFormatted(loader.String(), "web/main/plugins.go")
}

var configLoaderGeneratorTmpl = template.Must(template.New("configLoaderGenerator").Parse(
	`// generated by qs-build. do not edit.

package main

import (
	"{{.Import}}"
	"github.com/QuestScreen/qs-build/inspector"
)

func main() {
	err := inspector.InspectConfig("{{.ModuleName}}", {{.ModuleName}}.Descriptor.DefaultConfig)
	if err != nil {
		panic(err)
	}
}

`))

func executeInspector(pluginName, moduleName string) {
	ensureFileDoesntExistOrIsAutogenerated(
		filepath.Join(pluginName, moduleName, "web", "configloader.go"))

	dir, err := ioutil.TempDir("", "qs-build")
	if err != nil {
		logError("failed to create temporary directory:")
		logError(err.Error())
		os.Exit(1)
	}
	defer os.RemoveAll(dir)
	inspectorExe, err := os.Create(filepath.Join(dir, "main.go"))
	if err != nil {
		logError("failed to create file in temporary directory:")
		logError(err.Error())
		os.Exit(1)
	}
	if err = configLoaderGeneratorTmpl.Execute(inspectorExe, struct {
		Import, ModuleName string
	}{fmt.Sprintf("github.com/QuestScreen/QuestScreen/plugins/%v/%v",
		pluginName, moduleName), moduleName}); err != nil {
		inspectorExe.Close()
		logError("failed to generate content for config loader generator:")
		logError(err.Error())
		os.Exit(1)
	}
	inspectorExe.Close()
	cwd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(cwd)
	runAndDumpIfVerbose(exec.Command("go", "build", "-o", "main"),
		func(err error, stderr string) {
			logError(fmt.Sprintf("plugins/%v/%v [tmpdir: %v]:", pluginName, moduleName, dir))
			logError("failed to build inspector for module configuration:")
			logError(err.Error())
			writeErrorLines(stderr)
		})
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := exec.Command(filepath.Join(dir, "main"))
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err = cmd.Run(); err != nil {
		logError(fmt.Sprintf("plugins/%v/%v [tmpdir: %v]:", pluginName, moduleName, dir))
		logError("failed to execute inspector for module configuration:")
		logError(err.Error())
		writeErrorLines(stderr.String())
		os.Exit(1)
	}

	webDir := filepath.Join(cwd, pluginName, "web", moduleName)
	if err = os.Chdir(webDir); err != nil {
		logError(fmt.Sprintf("%v: unable to enter directory, missing?", webDir))
		os.Exit(1)
	}
	var loader strings.Builder
	loader.WriteString("// generated by qs-build. do not edit.\n\npackage ")
	loader.WriteString(moduleName)
	loader.WriteString("\n\n")
	loader.Write(stdout.Bytes())

	writeFormatted(loader.String(), "configloader.go")
}

func writeModuleConfigLoaders(plugins Data) {
	for _, plugin := range plugins {
		for _, module := range plugin.Modules {
			executeInspector(plugin.ID, module)
		}
	}
}

func writeModuleWebLoaders(plugins Data) {
	for _, plugin := range plugins {
		os.Chdir(plugin.ID)
		writeModuleLoader(plugin.ID, plugin.Modules)
		os.Chdir("..")
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
	writeModuleConfigLoaders(plugins)
	writeModuleWebLoaders(plugins)

	os.Chdir("..")
	writeWebPluginLoader(plugins)
}
