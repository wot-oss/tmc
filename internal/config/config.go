package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

const (
	KeyLog            = "log"
	KeyLogLevel       = "logLevel"
	KeyUrlContextRoot = "urlContextRoot"
	EnvPrefix         = "tmc"
)

var HomeDir string
var DefaultConfigDir string

func init() {
	var err error
	HomeDir, err = os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	DefaultConfigDir = filepath.Join(HomeDir, ".tm-catalog")

	InitViper()
}

func InitViper() {
	viper.SetDefault(KeyLog, false)
	viper.SetDefault(KeyLogLevel, "INFO")
	viper.SetDefault("remotes", map[string]any{
		"local": map[string]any{
			"type": "file",
			"loc":  "~/tm-catalog",
		},
	})

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
	// bind viper variable "log" to env (TMC_LOG)
	_ = viper.BindEnv(KeyLog)
	// bind viper variable "urlContextRoot" to env (TMC_URLCONTEXTROOT)
	_ = viper.BindEnv(KeyUrlContextRoot)

	viper.WatchConfig()
}
