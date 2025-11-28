package gogen

import (
	_ "embed"

	"github.com/blazy-vn/goctl/api/spec"
	"github.com/blazy-vn/goctl/config"
	"github.com/blazy-vn/goctl/internal/version"
	"github.com/blazy-vn/goctl/util/format"
)

//go:embed integration_test.tpl
var integrationTestTemplate string

func genIntegrationTest(dir, rootPkg, projectPkg string, cfg *config.Config, api *spec.ApiSpec) error {
	serviceName := api.Service.Name
	if len(serviceName) == 0 {
		serviceName = "server"
	}

	filename, err := format.FileNamingFormat(cfg.NamingFormat, serviceName)
	if err != nil {
		return err
	}

	return genFile(fileGenConfig{
		dir:             dir,
		subdir:          "",
		filename:        filename + "_test.go",
		templateName:    "integrationTestTemplate",
		category:        category,
		templateFile:    integrationTestTemplateFile,
		builtinTemplate: integrationTestTemplate,
		data: map[string]any{
			"projectPkg":  projectPkg,
			"serviceName": serviceName,
			"version":     version.BuildVersion,
			"hasRoutes":   len(api.Service.Routes()) > 0,
			"routes":      api.Service.Routes(),
		},
	})
}
