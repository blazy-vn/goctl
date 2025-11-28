package tanstackgen

import (
	"strings"

	"github.com/blazy-vn/goctl/api/spec"
)

func baseFileName(api *spec.ApiSpec) string {
	name := api.Service.Name
	if strings.HasSuffix(name, "-api") {
		return name[:len(name)-4]
	}
	return name
}

func typesFileName(api *spec.ApiSpec) string {
	return baseFileName(api) + "-types.ts"
}
