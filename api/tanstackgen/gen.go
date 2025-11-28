package tanstackgen

import (
	"errors"
	"fmt"

	"github.com/blazy-vn/goctl/api/parser"
	"github.com/blazy-vn/goctl/util/pathx"
	"github.com/gookit/color"
	"github.com/spf13/cobra"
	"github.com/zeromicro/go-zero/core/logx"
)

var (
	// VarStringDir describes a directory.
	VarStringDir string
	// VarStringAPI describes an API file.
	VarStringAPI string
)

// TanstackCommand provides the entry to generate TanStack Query/DB ready codes.
func TanstackCommand(_ *cobra.Command, _ []string) error {
	apiFile := VarStringAPI
	dir := VarStringDir
	if len(apiFile) == 0 {
		return errors.New("missing -api")
	}

	if len(dir) == 0 {
		return errors.New("missing -dir")
	}

	api, err := parser.Parse(apiFile)
	if err != nil {
		fmt.Println(color.Red.Render("Failed"))
		return err
	}

	if err := api.Validate(); err != nil {
		return err
	}

	api.Service = api.Service.JoinPrefix()
	logx.Must(pathx.MkdirIfNotExist(dir))
	logx.Must(genRequest(dir))
	logx.Must(genTypes(dir, api))
	logx.Must(genHandler(dir, api))

	fmt.Println(color.Green.Render("Done."))
	return nil
}
