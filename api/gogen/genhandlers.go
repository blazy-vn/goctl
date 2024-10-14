package gogen

import (
	"bytes"
	_ "embed"
	"fmt"
	"path"
	"strings"
	"text/template"

	"github.com/blazy-vn/goctl/api/spec"
	util2 "github.com/blazy-vn/goctl/api/util"
	"github.com/blazy-vn/goctl/config"
	"github.com/blazy-vn/goctl/util"
	"github.com/blazy-vn/goctl/util/format"
	"github.com/blazy-vn/goctl/util/pathx"
)

const defaultLogicPackage = "logic"

//go:embed handler.tpl
var handlerTemplate string

func genHandler(dir, rootPkg string, cfg *config.Config, group spec.Group, route spec.Route) error {
	handler := getHandlerName(route)
	handlerPath := getHandlerFolderPath(group, route)
	pkgName := handlerPath[strings.LastIndex(handlerPath, "/")+1:]
	logicName := defaultLogicPackage
	if handlerPath != handlerDir {
		handler = strings.Title(handler)
		logicName = pkgName
	}
	filename, err := format.FileNamingFormat(cfg.NamingFormat, handler)
	if err != nil {
		return err
	}

	return genFile(fileGenConfig{
		dir:             dir,
		subdir:          getHandlerFolderPath(group, route),
		filename:        filename + ".go",
		templateName:    "handlerTemplate",
		category:        category,
		templateFile:    handlerTemplateFile,
		builtinTemplate: handlerTemplate,
		data: map[string]any{
			"PkgName":        pkgName,
			"ImportPackages": genHandlerImports(group, route, rootPkg),
			"HandlerName":    handler,
			"RequestType":    util.Title(route.RequestTypeName()),
			"LogicName":      logicName,
			"LogicType":      util.Title(getLogicName(route)),
			"Call":           util.Title(strings.TrimSuffix(handler, "Handler")),
			"HasResp":        len(route.ResponseTypeName()) > 0,
			"HasRequest":     len(route.RequestTypeName()) > 0,
			"HasDoc":         len(route.JoinedDoc()) > 0,
			"Doc":            getDoc(route.JoinedDoc()),
		},
	})
}

func genHandlers(dir, rootPkg string, cfg *config.Config, api *spec.ApiSpec) error {
	authPath := path.Join(dir, authDir)

	if VarBoolAuth {
		if err := genAuthError(authPath, rootPkg, cfg, api); err != nil {
			return err
		}
	}

	if VarBoolAuth {
		if err := genPolicyFile(path.Join(dir, "etc"), rootPkg, cfg, api); err != nil {
			return err
		}
	}

	for _, group := range api.Service.Groups {
		if VarBoolAuth {
			if err := genAuth(authPath, rootPkg, cfg, group); err != nil {
				return err
			}
		}

		for _, route := range group.Routes {
			if err := genHandler(dir, rootPkg, cfg, group, route); err != nil {
				return err
			}
		}
	}

	return nil
}

func genHandlerImports(group spec.Group, route spec.Route, parentPkg string) string {
	imports := []string{
		fmt.Sprintf("\"%s\"", pathx.JoinPackages(parentPkg, getLogicFolderPath(group, route))),
		fmt.Sprintf("\"%s\"", pathx.JoinPackages(parentPkg, contextDir)),
	}
	if len(route.RequestTypeName()) > 0 {
		imports = append(imports, fmt.Sprintf("\"%s\"\n", pathx.JoinPackages(parentPkg, typesDir)))
	}

	return strings.Join(imports, "\n\t")
}

func getHandlerBaseName(route spec.Route) (string, error) {
	handler := route.Handler
	handler = strings.TrimSpace(handler)
	handler = strings.TrimSuffix(handler, "handler")
	handler = strings.TrimSuffix(handler, "Handler")

	return handler, nil
}

func getHandlerFolderPath(group spec.Group, route spec.Route) string {
	folder := route.GetAnnotation(groupProperty)
	if len(folder) == 0 {
		folder = group.GetAnnotation(groupProperty)
		if len(folder) == 0 {
			return handlerDir
		}
	}

	folder = strings.TrimPrefix(folder, "/")
	folder = strings.TrimSuffix(folder, "/")

	return path.Join(handlerDir, folder)
}

func getHandlerName(route spec.Route) string {
	handler, err := getHandlerBaseName(route)
	if err != nil {
		panic(err)
	}

	return handler + "Handler"
}

func getLogicName(route spec.Route) string {
	handler, err := getHandlerBaseName(route)
	if err != nil {
		panic(err)
	}

	return handler + "Logic"
}

func genAuth(dir, rootPkg string, cfg *config.Config, group spec.Group) error {
	authName := group.GetAnnotation(groupProperty)
	if len(authName) == 0 {
		return nil
	}

	authName = strings.TrimSuffix(authName, "s")
	authFilename := fmt.Sprintf("%s.go", strings.ToLower(authName))
	authName = util2.ToCamelCase(authName)

	authPkg := fmt.Sprintf("%s/auth", rootPkg)

	pkgParts := strings.Split(rootPkg, "/")
	moduleName := pkgParts[0]

	authImports := fmt.Sprintf(`"%s/common/bauth"
	"%s/ent"
	"context"
	"github.com/zeromicro/go-zero/core/logx"`, moduleName, moduleName)

	authMethods := make([]string, 0)
	for _, route := range group.Routes {
		handler := getHandlerName(route)
		method := strings.TrimSuffix(handler, "Handler")
		authMethods = append(authMethods, method)
	}

	authData := map[string]interface{}{
		"AuthPackage":    authPkg,
		"AuthName":       authName,
		"AuthImports":    authImports,
		"AuthMethods":    genAuthMethods(authName, authMethods),
		"AuthImplements": genAuthImplements(authName, authMethods),
	}

	return genFile(fileGenConfig{
		dir:             dir,
		filename:        authFilename,
		templateName:    "authTemplate",
		category:        category,
		templateFile:    "auth.tpl",
		builtinTemplate: authTemplate,
		data:            authData,
	})
}

//go:embed auth.tpl
var authTemplate string

//go:embed auth_error.tpl
var authErrorTemplate string

func genAuthMethods(authName string, authActions []string) string {
	var methods []string
	for _, action := range authActions {
		method := fmt.Sprintf(`Can%s(ctx context.Context, r *ent.%s) bool`, util.Title(action), authName)
		methods = append(methods, method)
	}
	return strings.Join(methods, "\n\t")
}

//go:embed auth_implement.tpl
var authImplementTemplate string

func genAuthImplements(authName string, authActions []string) string {
	var implements []string
	for _, action := range authActions {
		data := map[string]string{
			"AuthName":      authName,
			"Action":        util.Title(action),
			"AuthNameLower": strings.ToLower(util2.ToSnakeCase(authName)),
			"ActionLower":   strings.ToLower(util2.ToSnakeCase(action)),
		}
		var buf bytes.Buffer
		err := template.Must(template.New("authImplement").Parse(authImplementTemplate)).Execute(&buf, data)
		if err != nil {
			panic(err)
		}
		implements = append(implements, buf.String())
	}
	return strings.Join(implements, "\n\n")
}

func genAuthError(dir, rootPkg string, cfg *config.Config, api *spec.ApiSpec) error {
	pkgParts := strings.Split(rootPkg, "/")
	moduleName := pkgParts[0]

	errImports := fmt.Sprintf(`"%s/common/berr"`, moduleName)

	var errorVars []string
	for i, group := range api.Service.Groups {
		authName := group.GetAnnotation(groupProperty)
		if len(authName) == 0 {
			continue
		}

		authName = strings.TrimSuffix(authName, "s")
		authName = util2.ToCamelCase(authName)

		authMethods := make([]string, 0)
		for _, route := range group.Routes {
			handler := getHandlerName(route)
			method := strings.TrimSuffix(handler, "Handler")
			authMethods = append(authMethods, method)
		}

		baseErrCode := getBaseErrCode(i)
		errorVars = append(errorVars, genAuthErrorVars(authName, authMethods, baseErrCode))
	}

	authData := map[string]interface{}{
		"ErrImports": errImports,
		"ErrVars":    strings.Join(errorVars, "\n\n"),
	}

	return genFile(fileGenConfig{
		dir:             dir,
		filename:        "error.go",
		templateName:    "authErrorTemplate",
		category:        category,
		templateFile:    "", // Không cần thiết khi sử dụng embed
		builtinTemplate: authErrorTemplate,
		data:            authData,
	})
}

func genAuthErrorVars(authName string, authActions []string, baseErrCode int) string {
	var errorVars []string
	for i, action := range authActions {
		errCode := baseErrCode + i + 1
		errName := fmt.Sprintf("Err%s%sDenied", authName, strings.Title(action))
		errMsg := fmt.Sprintf("You do not have permission to perform this action: %s::%s",
			strings.ToLower(util2.ToSnakeCase(authName)),
			strings.ToLower(util2.ToSnakeCase(action)))
		errorVar := fmt.Sprintf("%s = berr.NewErrCodeMsg(%d, \"%s\")", errName, errCode, errMsg)
		errorVars = append(errorVars, errorVar)
	}
	return strings.Join(errorVars, "\n\t")
}

func genPolicyFile(dir, rootPkg string, cfg *config.Config, api *spec.ApiSpec) error {
	var policyLines []string
	for _, group := range api.Service.Groups {
		authName := group.GetAnnotation(groupProperty)
		if len(authName) == 0 {
			continue
		}

		authName = strings.ToLower(util2.ToSnakeCase(authName))
		policyLines = append(policyLines, fmt.Sprintf("p, %s_management, %s::*, true", authName, authName))

		for _, route := range group.Routes {
			handler := getHandlerName(route)
			method := strings.ToLower(util2.ToSnakeCase(strings.TrimSuffix(handler, "Handler")))
			policyLines = append(policyLines, fmt.Sprintf("p, %s_%s, %s::%s, true", authName, method, authName, method))
		}
	}

	policyData := strings.Join(policyLines, "\n")

	fp, _, err := util2.MaybeCreateFile(dir, "", "auth_policy.csv")
	if fp != nil {
		_, err = fp.WriteString(policyData)
	}

	return err
}

func getBaseErrCode(groupIndex int) int {
	return 2000 + groupIndex*100
}
