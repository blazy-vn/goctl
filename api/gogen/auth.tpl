package auth

import (
	"{{.importPath}}/common/bauth"
	"{{.importPath}}/ent"
	"context"
	"strings"
	"github.com/zeromicro/go-zero/core/logx"
)

type I{{.upperStartCamelObject}}Auth interface {
	{{range $func := .functions}}Can{{$func}}(ctx context.Context{{if ne $func "Add"}}, r ent.{{$upperStartCamelObject}}){{end}} bool
	{{end}}
}

type {{.upperStartCamelObject}}Auth struct {
	authSvc    *bauth.Authorizer
	identityFn func(ctx context.Context) string
}

{{range $func := .functions}}
func (a {{$upperStartCamelObject}}Auth) Can{{$func}}(ctx context.Context{{if ne $func "Add"}}, r ent.{{$upperStartCamelObject}}){{end}} bool {
	can, err := a.authSvc.Enforcer.Enforce(a.identityFn(ctx), {{if eq $func "Add"}}nil{{else}}r{{end}}, "{{$lowerObject}}::{{$funcLower}}")
	if err != nil {
		logx.WithContext(ctx).Errorf("enforce {{$lowerObject}}::{{$funcLower}} fail: %v", err)
		return false
	}
	return can
}
{{end}}

func New{{.upperStartCamelObject}}Auth(auth *bauth.Authorizer, iFn func(ctx context.Context) string) I{{.upperStartCamelObject}}Auth {
	return &{{.upperStartCamelObject}}Auth{authSvc: auth, identityFn: iFn}
}