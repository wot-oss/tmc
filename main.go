package main

import (
	"github.com/wot-oss/tmc/cmd"
	_ "github.com/wot-oss/tmc/cmd/repo"
	"github.com/wot-oss/tmc/internal"
	"github.com/wot-oss/tmc/internal/config"
)

func init() {
	config.InitConfig()
	config.InitViper()
	internal.InitLogging()
}
func main() {
	cmd.Execute()
}
