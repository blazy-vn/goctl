package tanstackgen

import (
	"strings"

	"github.com/blazy-vn/goctl/api/spec"
	apiutil "github.com/blazy-vn/goctl/api/util"
)

// BuildTypes generates the typescript code for the types.
func BuildTypes(types []spec.Type) (string, error) {
	var builder strings.Builder
	first := true
	for _, tp := range types {
		if first {
			first = false
		} else {
			builder.WriteString("\n")
		}
		if err := writeType(&builder, tp); err != nil {
			return "", apiutil.WrapErr(err, "Type "+tp.Name()+" generate error")
		}
	}

	return builder.String(), nil
}
