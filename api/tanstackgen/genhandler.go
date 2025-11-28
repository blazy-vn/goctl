package tanstackgen

import (
	_ "embed"
	"fmt"
	"path"
	"strings"
	"text/template"

	"github.com/blazy-vn/goctl/api/spec"
	apiutil "github.com/blazy-vn/goctl/api/util"
	"github.com/blazy-vn/goctl/util"
	"github.com/blazy-vn/goctl/util/pathx"
)

//go:embed handler.tpl
var handlerTemplate string

type routeParams struct {
	Type     string
	Required bool
}

type routeParts struct {
	HasBody    bool
	HasParams  bool
	HasHeader  bool
	ParamsType routeParams
	BodyType   string
	HeaderType string
}

func genHandler(dir string, api *spec.ApiSpec) error {
	filename := baseFileName(api) + ".ts"
	if err := pathx.RemoveIfExist(path.Join(dir, filename)); err != nil {
		return err
	}
	fp, created, err := apiutil.MaybeCreateFile(dir, "", filename)
	if err != nil {
		return err
	}
	if !created {
		return nil
	}
	defer fp.Close()

	imports := `import type { MutationOptions, QueryOptions } from "@tanstack/vue-query"`
	imports += fmt.Sprintf(`%simport { createMutationOptions, createQueryOptions, request, type ClientConfig } from "./tanstackRequest"`, pathx.NL)
	if len(api.Types) != 0 {
		outputFile := strings.TrimSuffix(typesFileName(api), ".ts")
		imports += fmt.Sprintf(`%simport * as types from "%s"`, pathx.NL, "./"+outputFile)
		imports += fmt.Sprintf(`%sexport * from "%s"`, pathx.NL, "./"+outputFile)
	}

	apis, err := genAPI(api)
	if err != nil {
		return err
	}

	t := template.Must(template.New("handlerTemplate").Parse(handlerTemplate))
	return t.Execute(fp, map[string]string{
		"imports": imports,
		"apis":    strings.TrimSpace(apis),
	})
}

func genAPI(api *spec.ApiSpec) (string, error) {
	var builder strings.Builder
	for _, group := range api.Service.Groups {
		for _, route := range group.Routes {
			routeCode, err := buildRoute(route, group)
			if err != nil {
				return "", err
			}
			builder.WriteString(routeCode)
			builder.WriteString("\n")
		}
	}

	return builder.String(), nil
}

func buildRoute(route spec.Route, group spec.Group) (string, error) {
	name, err := handlerNameForRoute(route, group)
	if err != nil {
		return "", err
	}

	responseType := "unknown"
	if len(route.ResponseTypeName()) > 0 {
		val, err := goTypeToTs(route.ResponseType, true)
		if err != nil {
			return "", err
		}
		responseType = val
	}

	parts, err := buildRouteParts(route)
	if err != nil {
		return "", err
	}

	variablesType := util.Title(name) + "Variables"
	var builder strings.Builder

	if comment := commentForRoute(route); len(comment) > 0 {
		builder.WriteString(comment)
		builder.WriteString("\n")
	}

	if len(route.RequestTypeName()) == 0 {
		builder.WriteString(fmt.Sprintf("export type %s = void\n\n", variablesType))
	} else {
		builder.WriteString(fmt.Sprintf("export type %s = {\n", variablesType))
		if parts.HasParams {
			builder.WriteString(fmt.Sprintf("\tparams%s: %s\n", optionalMarker(parts.ParamsType.Required), parts.ParamsType.Type))
		}
		if parts.HasBody {
			builder.WriteString(fmt.Sprintf("\tbody%s: %s\n", optionalMarker(true), parts.BodyType))
		}
		if parts.HasHeader {
			builder.WriteString(fmt.Sprintf("\theaders%s: %s\n", optionalMarker(true), parts.HeaderType))
		}
		builder.WriteString("}\n\n")
	}

	builder.WriteString(buildRequestFunction(name, route, group, responseType, parts, variablesType))
	builder.WriteString("\n")
	builder.WriteString(buildQueryOptionsFunction(name, route, group, responseType, parts, variablesType))
	builder.WriteString("\n")
	builder.WriteString(buildMutationOptionsFunction(name, responseType, parts, variablesType))
	builder.WriteString("\n")

	return builder.String(), nil
}

func optionalMarker(required bool) string {
	if required {
		return ""
	}
	return "?"
}

func buildRouteParts(route spec.Route) (routeParts, error) {
	result := routeParts{}
	ds, ok := resolveDefineStruct(route.RequestType)
	if !ok {
		return result, nil
	}

	result.HasBody = hasRequestBody(route)
	result.HasParams = hasRequestParams(route) || hasRequestPath(route)
	result.HasHeader = hasRequestHeader(route)

	if result.HasBody {
		bodyType, err := goTypeToTs(route.RequestType, true)
		if err != nil {
			return result, err
		}
		result.BodyType = bodyType
	}

	if result.HasHeader {
		headerType, err := goTypeToTs(route.RequestType, true)
		if err != nil {
			return result, err
		}
		result.HeaderType = headerType + "Headers"
	}

	if result.HasParams {
		paramsType, required, err := paramsTypeForRoute(ds, route)
		if err != nil {
			return result, err
		}
		result.ParamsType = routeParams{
			Type:     paramsType,
			Required: required,
		}
	}

	return result, nil
}

func paramsTypeForRoute(ds spec.DefineStruct, route spec.Route) (string, bool, error) {
	formMembers := ds.GetFormMembers()
	pathMembers := ds.GetTagMembers(pathTagKey)

	var formType string
	if len(formMembers) > 0 {
		val, err := goTypeToTs(route.RequestType, true)
		if err != nil {
			return "", false, err
		}
		formType = val + "Params"
	}

	pathType, err := pathParamsType(pathMembers)
	if err != nil {
		return "", false, err
	}

	switch {
	case formType != "" && pathType != "":
		return fmt.Sprintf("%s & %s", formType, pathType), true, nil
	case formType != "":
		return formType, true, nil
	case pathType != "":
		return pathType, true, nil
	default:
		return "", false, nil
	}
}

func pathParamsType(members []spec.Member) (string, error) {
	if len(members) == 0 {
		return "", nil
	}

	var builder strings.Builder
	builder.WriteString("{ ")
	for i, member := range members {
		tags := member.Tags()
		for _, tag := range tags {
			if tag.Key == pathTagKey {
				valueType, err := goTypeToTs(member.Type, false)
				if err != nil {
					return "", err
				}
				if i > 0 {
					builder.WriteString("; ")
				}
				builder.WriteString(fmt.Sprintf("%s: %s", tag.Name, valueType))
			}
		}
	}
	builder.WriteString(" }")
	return builder.String(), nil
}

func buildRequestFunction(name string, route spec.Route, group spec.Group, responseType string, parts routeParts, variablesType string) string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("export function %s(", name))
	var params []string
	if len(route.RequestTypeName()) > 0 {
		params = append(params, fmt.Sprintf("variables: %s", variablesType))
	}
	params = append(params, "config?: ClientConfig")
	builder.WriteString(strings.Join(params, ", "))
	builder.WriteString(") {\n")
	builder.WriteString(fmt.Sprintf("\treturn request<%s>({\n", responseType))
	builder.WriteString(fmt.Sprintf("\t\tpath: %s,\n", pathWithPrefix(route, group)))
	builder.WriteString(fmt.Sprintf("\t\tmethod: \"%s\",\n", strings.ToUpper(route.Method)))
	if parts.HasParams {
		builder.WriteString("\t\tparams: variables.params,\n")
	}
	if parts.HasBody {
		builder.WriteString("\t\tbody: variables.body,\n")
	}
	if parts.HasHeader {
		builder.WriteString("\t\theaders: variables.headers,\n")
	}
	builder.WriteString("\t}, config)\n")
	builder.WriteString("}\n")
	return builder.String()
}

func buildQueryOptionsFunction(name string, route spec.Route, group spec.Group, responseType string, parts routeParts, variablesType string) string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("export function %sQueryOptions(", name))
	var params []string
	if len(route.RequestTypeName()) > 0 {
		params = append(params, fmt.Sprintf("variables: %s", variablesType))
	}
	params = append(params, fmt.Sprintf("options?: Partial<QueryOptions<%s, Error>>", responseType))
	params = append(params, "config?: ClientConfig")
	builder.WriteString(strings.Join(params, ", "))
	builder.WriteString(fmt.Sprintf("): QueryOptions<%s, Error> {\n", responseType))

	keyTail := queryKeyTail(parts)
	if len(route.RequestTypeName()) > 0 && len(keyTail) == 0 {
		keyTail = ", variables"
	}
	queryKey := fmt.Sprintf("['%s'%s] as const", name, keyTail)
	builder.WriteString(fmt.Sprintf("\treturn createQueryOptions<%s>(\n", responseType))
	builder.WriteString(fmt.Sprintf("\t\t%s,\n", queryKey))
	builder.WriteString("\t\t({ signal }) => ")
	builder.WriteString(fmt.Sprintf("%s(", name))
	if len(route.RequestTypeName()) > 0 {
		builder.WriteString("variables, ")
	}
	builder.WriteString("config ? { ...config, signal } : { signal }")
	builder.WriteString("),\n")
	builder.WriteString("\t\toptions,\n")
	builder.WriteString("\t)\n")
	builder.WriteString("}\n")
	return builder.String()
}

func buildMutationOptionsFunction(name, responseType string, parts routeParts, variablesType string) string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("export function %sMutationOptions(", name))
	params := []string{
		fmt.Sprintf("options?: Partial<MutationOptions<%s, Error, %s>>", responseType, variablesType),
		"config?: ClientConfig",
	}
	builder.WriteString(strings.Join(params, ", "))
	builder.WriteString(fmt.Sprintf("): MutationOptions<%s, Error, %s> {\n", responseType, variablesType))
	builder.WriteString(fmt.Sprintf("\treturn createMutationOptions<%s, %s>(\n", responseType, variablesType))
	// Always pass variables parameter, even for empty request types
	// The function signature expects (variables, config?) not (config)
	builder.WriteString(fmt.Sprintf("\t\t(variables) => %s(variables, config),\n", name))
	builder.WriteString("\t\toptions,\n")
	builder.WriteString("\t)\n")
	builder.WriteString("}\n")
	return builder.String()
}

func queryKeyTail(parts routeParts) string {
	var segments []string
	if parts.HasParams {
		segments = append(segments, "variables?.params")
	}
	if parts.HasBody {
		segments = append(segments, "variables?.body")
	}
	if parts.HasHeader {
		segments = append(segments, "variables?.headers")
	}
	if len(segments) == 0 {
		return ""
	}
	return ", " + strings.Join(segments, ", ")
}

func pathWithPrefix(route spec.Route, group spec.Group) string {
	prefix := group.GetAnnotation(pathPrefix)
	routePath := route.Path
	if len(prefix) == 0 {
		return "`" + routePath + "`"
	}

	prefix = strings.Trim(prefix, `"`)
	return fmt.Sprintf("`%s/%s`", strings.TrimSuffix(prefix, "/"), strings.TrimPrefix(routePath, "/"))
}
