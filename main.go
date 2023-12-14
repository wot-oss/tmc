/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"github.com/web-of-things-open-source/tm-catalog-cli/cmd"
	_ "github.com/web-of-things-open-source/tm-catalog-cli/cmd/remote"
)

func main() {
	cmd.Execute()
}
