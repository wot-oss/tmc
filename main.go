/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"github.com/spf13/viper"
	"github.com/web-of-things-open-source/tm-catalog-cli/cmd"
	_ "github.com/web-of-things-open-source/tm-catalog-cli/cmd/remote"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/config"
)

func main() {
	cmd.Execute()
}

func init() {
	initViper()
}

func initViper() {
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

	// set prefix "tmc" for environment variables
	// the environment variables then have to match pattern "tmc_<viper variable>", lower or uppercase
	viper.SetEnvPrefix("tmc")
	// bind viper variable "log" to env (tmc_log or TMC_LOG)
	_ = viper.BindEnv("log")
	// bind viper variable "log" also to CLI flag --log of root command
	_ = viper.BindPFlag("log", cmd.RootCmd.PersistentFlags().Lookup("log"))

	viper.WatchConfig()
}
