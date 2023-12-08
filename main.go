/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"github.com/spf13/cobra"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/config"
	"io"
	"log/slog"
	"os"

	"github.com/spf13/viper"
	"github.com/web-of-things-open-source/tm-catalog-cli/cmd"
	_ "github.com/web-of-things-open-source/tm-catalog-cli/cmd/remote"
)

func main() {
	cmd.Execute()
}

func init() {
	cobra.OnInitialize(configureCLI)
}

func configureCLI() {
	configureViper()
	configureLogger()
}

func configureViper() {
	viper.SetDefault("log", false)
	viper.SetDefault("logLevel", "INFO")
	viper.SetDefault("remotes", map[string]any{
		"local": map[string]any{
			"type": "file",
			"loc":  "~/tm-catalog",
		},
	})

	viper.SetConfigType("json")
	viper.SetConfigName("config")
	viper.AddConfigPath(config.DefaultConfigDir)
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

	_ = viper.BindEnv("log")
	_ = viper.BindPFlag("log", cmd.RootCmd.PersistentFlags().Lookup("log"))

	viper.WatchConfig()
}

func configureLogger() {
	logEnabled := viper.GetBool("log")
	var writer = io.Discard
	if logEnabled {
		writer = os.Stderr
	}

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

	handler := slog.NewTextHandler(writer, opts)
	log := slog.New(handler)
	slog.SetDefault(log)
}
