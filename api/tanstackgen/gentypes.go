package tanstackgen

import (
	_ "embed"
	"path"
	"strings"
	"text/template"

	"github.com/blazy-vn/goctl/api/spec"
	apiutil "github.com/blazy-vn/goctl/api/util"
	"github.com/blazy-vn/goctl/internal/version"
	"github.com/blazy-vn/goctl/util/pathx"
)

//go:embed components.tpl
var componentsTemplate string

func genTypes(dir string, api *spec.ApiSpec) error {
	types := api.Types
	if len(types) == 0 {
		return nil
	}

	val, err := BuildTypes(types)
	if err != nil {
		return err
	}

	outputFile := typesFileName(api)
	filename := path.Join(dir, outputFile)
	if err := pathx.RemoveIfExist(filename); err != nil {
		return err
	}

	fp, created, err := apiutil.MaybeCreateFile(dir, ".", outputFile)
	if err != nil {
		return err
	}
	if !created {
		return nil
	}
	defer fp.Close()

	t := template.Must(template.New("componentsTemplate").Parse(componentsTemplate))
	return t.Execute(fp, map[string]string{
		"componentTypes": strings.TrimSpace(val),
		"version":        version.BuildVersion,
	})
}
