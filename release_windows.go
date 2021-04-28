//+build windows

package main

import (
	"archive/zip"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func findDll(name string) string {
	path := runAndCheck(exec.Command("where", name+".dll"), func(err error, stderr string) {
		logError("unable to locate " + name + ":")
		logError(err.Error())
	})
	return strings.TrimSpace(path)
}

func releaseWindowsBinary(relname string) {
	for i := range commands {
		logPhase(commands[i].name)
		commands[i].exec()
	}
	logPhase("Release")

	logInfo("finding DLLs")
	libs := [3]string{
		findDll("SDL2"),
		findDll("SDL2_image"),
		findDll("SDL2_ttf"),
	}

	logInfo("creating " + relname + ".zip")
	relzip, err := os.Create(relname + ".zip")
	must(err)
	defer relzip.Close()

	w := zip.NewWriter(relzip)
	defer w.Close()

	addFiles := func(base, root string) {
		must(filepath.Walk(root, func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			inclPath, err := filepath.Rel(base, path)
			must(err)

			zf, err := w.Create(filepath.Join(relname, inclPath))
			if err != nil {
				return err
			}
			_, err = io.Copy(zf, file)
			if err != nil {
				return err
			}

			return nil
		}))
	}
	for _, lib := range libs {
		addFiles(filepath.Dir(lib), lib)
	}
	addFiles(".", "questscreen.exe")
	addFiles(".", "resources")
}
