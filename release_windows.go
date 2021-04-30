//+build windows

package main

import (
	"archive/zip"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
)

func findDll(name string) string {
	path := runAndCheck(exec.Command("where", name+".dll"), func(err error, stderr string) {
		logError("unable to locate " + name + ":")
		logError(err.Error())
	})
	return strings.TrimSpace(path)
}

func createFullscreenLink(path string) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED|ole.COINIT_SPEED_OVER_MEMORY)
	defer ole.CoUninitialize()

	oleShellObject, err := oleutil.CreateObject("WScript.Shell")
	if err != nil {
		return err
	}
	defer oleShellObject.Release()
	wshell, err := oleShellObject.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return err
	}
	defer wshell.Release()
	cs, err := oleutil.CallMethod(wshell, "CreateShortcut", path)
	if err != nil {
		return err
	}
	idispatch := cs.ToIDispatch()
	oleutil.PutProperty(idispatch, "TargetPath", "%COMSPEC%")
	oleutil.PutProperty(idispatch, "Arguments", "/C .\\questscreen.exe -f")
	oleutil.CallMethod(idispatch, "Save")
	return nil
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

	logInfo("summoning OLE from the depths of hell to create fullscreen link")
	must(createFullscreenLink("questscreen - fullscreen.lnk"))
	defer os.Remove("questscreen - fullscreen.lnk")

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
	addFiles(".", "questscreen - fullscreen.lnk")
	addFiles(".", "resources")
}
