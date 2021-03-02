package inspector

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"text/template"

	"github.com/QuestScreen/api/config"
)

type configField struct {
	Label, ConstructorName string
}

type importData struct {
	Name, Path string
}

type configData struct {
	Items   []configField
	Imports []importData
}

var tmpl = template.Must(template.New("configLoader").Parse(`
import (
	"github.com/QuestScreen/api/web/modules"
	"github.com/QuestScreen/api/web/config"
	"encoding/json"
	{{- range .Imports}}
	{{.Name}} "{{.Path}}"
	{{- end}}
)

var ConfigDescriptor = []modules.ConfigItem{
	{{- range .Items}}
	{
		Constructor: {{.ConstructorName}},
		Name: "{{.Label}}",
	},
	{{- end}}
}
`))

// InspectConfig uses reflection to inspect a default configuration value
// and writes code for loading a configuration with that value to stdout.
func InspectConfig(modName string, confValue interface{}) error {
	rValue := reflect.ValueOf(confValue)
	rType := rValue.Type()
	if rType.Kind() == reflect.Ptr {
		rValue = rValue.Elem()
		rType = rValue.Type()
	}
	if rType.Kind() != reflect.Struct {
		return fmt.Errorf("module %v: default configuration is not a struct", modName)
	}

	var data configData
	for i := 0; i < rType.NumField(); i++ {
		field := rType.Field(i)
		fVal := rValue.Field(i)
		fType := field.Type
		if fType.Kind() != reflect.Ptr {
			return fmt.Errorf("module %v: config field %v does not have pointer type", modName, field.Name)
		}
		fType = fType.Elem()
		if _, ok := fVal.Interface().(config.Item); !ok {
			return fmt.Errorf("module %v: config field %v is not a config.Item", modName, field.Name)
		}
		pkgPathElms := strings.Split(fType.PkgPath(), "/")
		webPath := strings.Join(append(pkgPathElms[:len(pkgPathElms)-1], "web",
			pkgPathElms[len(pkgPathElms)-1]), "/")

		var importD importData
		for _, item := range data.Imports {
			if item.Path == webPath {
				importD = item
				break
			}
		}
		if importD.Name == "" {
			importD = importData{Name: fmt.Sprintf("p%v", len(data.Imports)+1),
				Path: webPath}
			data.Imports = append(data.Imports, importD)
		}
		item := configField{Label: field.Name, ConstructorName: importD.Name + ".New" + fType.Name()}
		data.Items = append(data.Items, item)
	}
	if err := tmpl.Execute(os.Stdout, data); err != nil {
		return fmt.Errorf("module %v: %v", modName, err.Error())
	}
	return nil
}
