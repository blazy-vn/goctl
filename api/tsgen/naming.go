package tsgen

import (
	"strings"

	"github.com/blazy-vn/goctl/api/spec"
	"github.com/blazy-vn/goctl/util"
	"github.com/blazy-vn/goctl/util/stringx"
)

func baseFileName(api *spec.ApiSpec) string {
	name := api.Service.Name
	if strings.HasSuffix(name, "-api") {
		return name[:len(name)-4]
	}
	return name
}

// groupFileBase returns a safe file base name for a group.
// If group annotation is missing, fall back to the service name.
func groupFileBase(api *spec.ApiSpec, group spec.Group) string {
	name := group.GetAnnotation(groupProperty)
	if len(name) == 0 {
		return baseFileName(api)
	}

	name = strings.Trim(name, "\"")
	name = strings.Trim(name, "/")
	if len(name) == 0 {
		return baseFileName(api)
	}

	name = stringx.From(name).ToSnake()
	return util.SafeString(name)
}
