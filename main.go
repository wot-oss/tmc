/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"github.com/spf13/viper"
	"github.com/web-of-things-open-source/tm-catalog-cli/cmd"
	"log/slog"
	"os"
)

func main() {
	cmd.Execute()
}

func init() {
	initViper()
	setUpLogger()
}

func initViper() {
	viper.SetDefault("remotes", map[string]any{
		"localFS": map[string]any{
			"type": "file",
			"url":  "file:~/tm-catalog",
		},
	})
	viper.SetDefault("logLevel", "INFO")

	viper.SetConfigType("json")
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		if err := viper.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); ok {
				// Config file not found; do nothing and rely on defaults
			} else {
				panic("cannot read config: " + err.Error())
			}
		}
	}
}
func setUpLogger() {
	logLevel := viper.GetString("logLevel")
	var level slog.Level
	//levelP := &level
	err := level.UnmarshalText([]byte(logLevel))
	if err != nil {
		level = slog.LevelInfo
	}
	opts := &slog.HandlerOptions{
		Level: level,
	}
	handler := slog.NewTextHandler(os.Stderr, opts)
	log := slog.New(handler)
	slog.SetDefault(log)
}
