package tsgen

import (
	"testing"

	"github.com/blazy-vn/goctl/api/spec"
	"github.com/stretchr/testify/assert"
)

func TestHandlerNameForRoute_RequestFallback(t *testing.T) {
	group := spec.Group{}

	route1 := spec.Route{
		Handler:     "List",
		Path:        "/v1/bill-item/list",
		RequestType: spec.DefineStruct{RawName: "BillItemListRequest"},
	}
	route2 := spec.Route{
		Handler:     "List",
		Path:        "/v1/bundle/list",
		RequestType: spec.DefineStruct{RawName: "BundleListRequest"},
	}

	name1, err := handlerNameForRoute(route1, group)
	assert.NoError(t, err)
	assert.Equal(t, "list", name1)

	name2, err := handlerNameForRoute(route2, group)
	assert.NoError(t, err)
	assert.Equal(t, "list", name2)
}

func TestHandlerNameForRoute_PathFallback(t *testing.T) {
	group := spec.Group{Annotation: spec.Annotation{Properties: map[string]string{pathPrefix: `"/v1"`}}}

	route1 := spec.Route{Handler: "Detail", Path: "/bill-item/:id"}
	route2 := spec.Route{Handler: "Detail", Path: "/bundle/:id"}

	name1, err := handlerNameForRoute(route1, group)
	assert.NoError(t, err)
	assert.Equal(t, "detail", name1)

	name2, err := handlerNameForRoute(route2, group)
	assert.NoError(t, err)
	assert.Equal(t, "detail", name2)
}

func TestHandlerNameForRoute_GroupFallback(t *testing.T) {
	group1 := spec.Group{Annotation: spec.Annotation{Properties: map[string]string{groupProperty: "department"}}}
	group2 := spec.Group{Annotation: spec.Annotation{Properties: map[string]string{groupProperty: "doctor-team"}}}

	route1 := spec.Route{Handler: "ListHandler", Path: "/list"}
	route2 := spec.Route{Handler: "ListHandler", Path: "/list"}

	name1, err := handlerNameForRoute(route1, group1)
	assert.NoError(t, err)
	assert.Equal(t, "departmentList", name1)

	name2, err := handlerNameForRoute(route2, group2)
	assert.NoError(t, err)
	assert.Equal(t, "doctorTeamList", name2)
}
