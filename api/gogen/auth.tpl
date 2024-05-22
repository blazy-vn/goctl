// auth.tpl
package auth

import (
	{{.AuthImports}}
)

type I{{.AuthName}}Auth interface {
	{{.AuthMethods}}
}

type {{.AuthName}}Auth struct {
	authSvc    *bauth.Authorizer
	identityFn func(ctx context.Context) string
}

{{.AuthImplements}}

func New{{.AuthName}}Auth(auth *bauth.Authorizer, iFn func(ctx context.Context) string) I{{.AuthName}}Auth {
	return &{{.AuthName}}Auth{authSvc: auth, identityFn: iFn}
}