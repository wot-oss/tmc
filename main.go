/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"github.com/web-of-things-open-source/tm-catalog-cli/cmd"
	"log/slog"
	"os"
)

func main() {
	cmd.Execute()
}

func init() {
	setUpLogger()
}
func setUpLogger() {
	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug, // fixme: read from viper config
	}
	handler := slog.NewTextHandler(os.Stdout, opts)
	log := slog.New(handler)
	slog.SetDefault(log)
}
