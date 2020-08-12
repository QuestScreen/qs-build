package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

func discoverPlugins() map[string]string {
	os.Stdout.WriteString("[info] discovering plugins\n")
	plugindirs, err := ioutil.ReadDir(".")
	if err != nil {
		panic(err)
	}
	plugins := make(map[string]string)
	for _, plugindir := range plugindirs {
		if !plugindir.IsDir() {
			continue
		}
		path := filepath.Join(plugindir.Name(), "questscreen-plugin.txt")
		info, err := os.Stat(path)
		if err != nil {
			os.Stdout.WriteString("[warning] " + path + ": " + err.Error() + "\n")
			os.Stdout.WriteString("[warning] " + plugindir.Name() + ": skipping\n")
			continue
		}
		if info.IsDir() {
			os.Stdout.WriteString("[warning] " + path + ": is a directory\n")
			os.Stdout.WriteString("[warning] " + plugindir.Name() + ": skipping\n")
			continue
		}
		varName, err := ioutil.ReadFile(path)
		if err != nil {
			os.Stdout.WriteString("[warning] " + path + ": " + err.Error() + "\n")
			os.Stdout.WriteString("[warning] " + plugindir.Name() + ": skipping\n")
			continue
		}
		plugins[plugindir.Name()] = string(varName)
	}
	return plugins
}

func writePluginCollector(plugins map[string]string) {
	os.Stdout.WriteString("[info] generating plugins/plugins.go\n")
	var loader strings.Builder
	loader.WriteString("package plugins\n\nimport(\n")
	loader.WriteString("\t\"github.com/QuestScreen/QuestScreen/app\"\n")
	for dir := range plugins {
		loader.WriteString("\t\"github.com/QuestScreen/QuestScreen/plugins/" + dir + "\"\n")
	}
	loader.WriteString(")\n\nfunc LoadPlugins(a app.App) {\n")
	for dir, name := range plugins {
		loader.WriteString("\ta.AddPlugin(&" + dir + "." + name + ")\n")
	}
	loader.WriteString("}\n")
	writeFormatted(loader.String(), "plugins.go")
}

func writeWebPluginLoader(plugins map[string]string) {
	os.Stdout.WriteString("[info] generating web/main/plugins.go\n")
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
	writePluginCollector(plugins)

	os.Chdir("..")
	writeWebPluginLoader(plugins)
}
