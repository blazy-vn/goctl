package upgrade

import "github.com/blazy-vn/go-zero/tools/goctl/internal/cobrax"

// Cmd describes an upgrade command.
var Cmd = cobrax.NewCommand("upgrade", cobrax.WithRunE(upgrade))
