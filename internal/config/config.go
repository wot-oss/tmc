package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

const (
	KeyLog                  = "log"
	KeyLogLevel             = "logLevel"
	KeyUrlContextRoot       = "urlContextRoot"
	KeyCorsAllowedOrigins   = "corsAllowedOrigins"
	KeyCorsAllowedHeaders   = "corsAllowedHeaders"
	KeyCorsAllowCredentials = "corsAllowCredentials"
	EnvPrefix               = "tmc"
)

var HomeDir string
var DefaultConfigDir string

func InitConfig() {
	var err error
	HomeDir, err = os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	DefaultConfigDir = filepath.Join(HomeDir, ".tm-catalog")

}

func InitViper() {
	viper.SetDefault("remotes", map[string]any{})
	viper.SetDefault(KeyLog, false)
	viper.SetDefault(KeyLogLevel, "INFO")

	viper.SetConfigType("json")
	viper.SetConfigName("config")
	viper.AddConfigPath(DefaultConfigDir)
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; do nothing and rely on defaults
		} else {
			panic("cannot read config: " + err.Error())
		}
	}
	// set prefix "tmc" for environment variables
	// the environment variables then have to match pattern "tmc_<viper variable>", lower or uppercase
	viper.SetEnvPrefix(EnvPrefix)

	// bind viper variables to environment variables
	_ = viper.BindEnv(KeyLog)                  // env variable name = TMC_LOG
	_ = viper.BindEnv(KeyUrlContextRoot)       // env variable name = TMC_URLCONTEXTROOT
	_ = viper.BindEnv(KeyCorsAllowedOrigins)   // env variable name = TMC_CORSALLOWEDORIGINS
	_ = viper.BindEnv(KeyCorsAllowedHeaders)   // env variable name = TMC_CORSALLOWEDHEADERS
	_ = viper.BindEnv(KeyCorsAllowCredentials) // env variable name = TMC_CORSALLOWCREDENTIALS
}
