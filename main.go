package main

import (
	"github.com/web-of-things-open-source/tm-catalog-cli/cmd"
	_ "github.com/web-of-things-open-source/tm-catalog-cli/cmd/repo"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/config"
)

func init() {
	config.InitConfig()
	config.InitViper()
	internal.InitLogging()
}
func main() {
	cmd.Execute()
}
