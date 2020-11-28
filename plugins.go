package main

import (
	"errors"
	"io/ioutil"
	"os"
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

var pluginsGoTmpl = template.Must(template.New("plugins.go").Funcs(template.FuncMap{
	"yaml": func(value interface{}) (string, error) {
		out, err := yaml.Marshal(value)
		if err != nil {
			return "", err
		}
		return string(out), nil
	},
	"searchPlugin": func(root map[string]PluginDescr, path string) (int, error) {
		items := strings.Split(path, ".")
		index := 0
		for key := range root {
			if key == items[0] {
				return index, nil
			}
			index = index + 1
		}
		return 0, errors.New("cannot resolve scene template reference, unknown plugin: " + items[0])
	},
	"searchTemplate": func(root map[string]PluginDescr, path string) (int, error) {
		items := strings.Split(path, ".")
		for key, value := range root {
			if key == items[0] {
				index := 0
				for scene := range value.Templates.Scenes {
					if scene == items[1] {
						return index, nil
					}
					index = index + 1
				}
				return 0, errors.New("cannot resolve scene template reference, unknown scene: " + path)
			}
		}
		return 0, errors.New("cannot resolve scene template reference, unknown plugin: " + path)
	},
}).Parse(`{{$root := . -}}
package plugins

import (
	"github.com/QuestScreen/api/modules"
	"github.com/QuestScreen/QuestScreen/app"
	{{- range $key, $value := .}}
	"github.com/QuestScreen/QuestScreen/plugins/{{$key}}"
	{{- range $value.Modules}}
	"github.com/QuestScreen/QuestScreen/plugins/{{$key}}/{{.}}"
	{{- end}}
	{{- end}}
)

func LoadPlugins(a app.App) {
	{{range $key, $value := . -}}
	a.AddPlugin("{{$key}}", &app.Plugin{
		Name: "{{js $value.Name}}",
		Modules: []*modules.Module{
			{{- range $value.Modules}}
			&{{.}}.Descriptor,
			{{- end}}
		},
		SystemTemplates: []app.SystemTemplate{
			{{- range $ID, $content := $value.Templates.Systems}}
			{
				ID: "{{js $ID}}",
				Name: "{{js $content.Name}}",
				Config: []byte(` + "`" + `{{yaml $content.Config}}` + "`" + `),
			},
			{{- end}}
		},
		GroupTemplates: []app.GroupTemplate{
			{{- range $value.Templates.Groups}}
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
			{{- range $value.Templates.Scenes}}
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

func writePluginCollector(plugins map[string]PluginDescr) {
	logInfo("generating plugins/plugins.go")
	var writer strings.Builder
	pluginsGoTmpl.Execute(&writer, plugins)
	writeFormatted(writer.String(), "plugins.go")
}

func writeWebPluginLoader(plugins map[string]PluginDescr) {
	logInfo("generating web/main/plugins.go")
	var loader strings.Builder
	loader.WriteString("package main\n\nimport(\n")
	for dir := range plugins {
		loader.WriteString("\t" + dir + " \"github.com/QuestScreen/QuestScreen/plugins/" + dir + "/web\"\n")
	}
	loader.WriteString(")\n\nfunc loadPlugins(loader *pluginLoader) {\n")
	for dir := range plugins {
		loader.WriteString("\t" + dir + ".LoadPluginWebUI(loader)\n")
	}
	loader.WriteString("}\n")
	writeFormatted(loader.String(), "web/main/plugins.go")
}

func writePluginLoaders() {
	os.Chdir("plugins")
	plugins := discoverPlugins()

	for _, plugin := range plugins {
		for _, value := range plugin.Templates.Systems {
			value.Config["name"] = value.Name
		}
	}

	writePluginCollector(plugins)

	os.Chdir("..")
	writeWebPluginLoader(plugins)
}
