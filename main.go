package main

import (
	"github.com/wot-oss/tmc/cmd"
	_ "github.com/wot-oss/tmc/cmd/attachment"
	_ "github.com/wot-oss/tmc/cmd/repo"
)

func main() {
	cmd.Execute()
}
