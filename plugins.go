package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

func discoverPlugins() map[string]string {
	logInfo("discovering plugins")
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
			logWarning(path + ": " + err.Error())
			logWarning(plugindir.Name() + ": skipping")
			continue
		}
		if info.IsDir() {
			logWarning(path + ": is a directory")
			logWarning(plugindir.Name() + ": skipping")
			continue
		}
		varName, err := ioutil.ReadFile(path)
		if err != nil {
			logWarning(path + ": " + err.Error())
			logWarning(plugindir.Name() + ": skipping")
			continue
		}
		plugins[plugindir.Name()] = string(varName)
	}
	return plugins
}

func writePluginCollector(plugins map[string]string) {
	logInfo("generating plugins/plugins.go")
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
	writePluginCollector(plugins)

	os.Chdir("..")
	writeWebPluginLoader(plugins)
}
