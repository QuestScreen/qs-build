package inspector

import (
	"fmt"
	"os"
	"reflect"
	"text/template"

	"github.com/QuestScreen/api/config"
)

type configField struct {
	Label, ControllerName string
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
	"github.com/QuestScreen/api/web/server"
	"github.com/QuestScreen/api/web/config"
	"encoding/json"
	{{- range .Imports}}
	{{.Name}} "{{.Path}}/web"
	{{- end}}
)

// LoadConfig loads configuration values of this module.
func LoadConfig(ctx server.Context, input []byte) (map[string]config.Controller, error) {
	var items map[string]json.RawMessage
	if err := json.Unmarshal(input, items); err != nil {
		return nil, err
	}
	ret := make(map[string]config.Controller)
	{{- range .Items}}
	{
		raw, ok := items["{{.Label}}"]
		if !ok {
			return nil, errors.New("field missing: " + "{{.Label}}")
		}
		var value {{.ControllerName}}
		if err := json.Unmarshal(raw, &value); err != nil {
			return nil, errors.New("in config item {{.Label}}: " + err.Error())
		}
		ret["{{.Label}}"] = &value
		delete(items, "{{.Label}}")
	}
	{{- end}}
	for key := range items {
		return nil, errors.New("unknown field: " + key)
	}
	return ret, nil
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
		var importD importData
		for _, item := range data.Imports {
			if item.Path == fType.PkgPath() {
				importD = item
				break
			}
		}
		if importD.Name == "" {
			importD = importData{Name: fmt.Sprintf("p%v", len(data.Imports)+1),
				Path: fType.PkgPath()}
			data.Imports = append(data.Imports, importD)
		}
		item := configField{Label: field.Name, ControllerName: importD.Name + ".Controller"}
		data.Items = append(data.Items, item)
	}
	if err := tmpl.Execute(os.Stdout, data); err != nil {
		return fmt.Errorf("module %v: %v", modName, err.Error())
	}
	return nil
}
