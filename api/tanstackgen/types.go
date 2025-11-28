package tanstackgen

import (
	"strings"

	"github.com/blazy-vn/goctl/api/spec"
	apiutil "github.com/blazy-vn/goctl/api/util"
)

// BuildTypes generates the typescript code for the types.
func BuildTypes(types []spec.Type, sharedTypes map[string]struct{}, inShared bool) (string, bool, error) {
	var builder strings.Builder
	var needsShared bool
	first := true
	for _, tp := range types {
		if first {
			first = false
		} else {
			builder.WriteString("\n")
		}
		if err := writeType(&builder, tp, sharedTypes, inShared, &needsShared); err != nil {
			return "", false, apiutil.WrapErr(err, "Type "+tp.Name()+" generate error")
		}
	}

	return builder.String(), needsShared, nil
}
