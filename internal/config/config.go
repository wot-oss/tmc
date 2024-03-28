package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/wot-oss/tmc/internal/utils"

	"github.com/spf13/viper"
)

const (
	KeyLogLevel             = "logLevel"
	KeyUrlContextRoot       = "urlContextRoot"
	KeyCorsAllowedOrigins   = "corsAllowedOrigins"
	KeyCorsAllowedHeaders   = "corsAllowedHeaders"
	KeyCorsAllowCredentials = "corsAllowCredentials"
	KeyCorsMaxAge           = "corsMaxAge"
	KeyJWTValidation        = "jwtValidation"
	KeyJWTServiceID         = "jwtServiceID"
	KeyJWKSURL              = "jwksURL"
	EnvPrefix               = "tmc"
	LogLevelOff             = "off"

	modSet int = iota
	modDel
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
	//viper.SetDefault("repos", map[string]any{})
	viper.SetDefault(KeyLogLevel, LogLevelOff)
	viper.SetDefault(KeyJWTValidation, false)

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
	_ = viper.BindEnv(KeyLogLevel)             // env variable name = tmc_loglevel
	_ = viper.BindEnv(KeyUrlContextRoot)       // env variable name = tmc_urlcontextroot
	_ = viper.BindEnv(KeyCorsAllowedOrigins)   // env variable name = tmc_corsallowedorigins
	_ = viper.BindEnv(KeyCorsAllowedHeaders)   // env variable name = tmc_corsallowedheaders
	_ = viper.BindEnv(KeyCorsAllowCredentials) // env variable name = tmc_corsallowcredentials
	_ = viper.BindEnv(KeyCorsMaxAge)           // env variable name = tmc_corsmaxage
	_ = viper.BindEnv(KeyJWTValidation)        // env variable name = tmc_jwtvalidation
	_ = viper.BindEnv(KeyJWTServiceID)         // env variable name = tmc_jwtvalidation
	_ = viper.BindEnv(KeyJWKSURL)              // env variable name = tmc_jwksurl
}

func SaveConfig(key string, data any) error {
	viper.Set(key, data)
	return updateConfigFile(modSet, key, data)
}

func DeleteConfig(key string) error {
	return updateConfigFile(modDel, key, nil)
}

func updateConfigFile(mod int, key string, data any) error {

	configFile := viper.ConfigFileUsed()
	if configFile == "" {
		configFile = filepath.Join(DefaultConfigDir, "config.json")
	}
	err := os.MkdirAll(DefaultConfigDir, 0770)
	if err != nil {
		return err
	}

	b, err := os.ReadFile(configFile)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if len(b) == 0 {
		b = []byte("{}")
	}
	var j map[string]any
	err = json.Unmarshal(b, &j)
	if err != nil {
		return err
	}
	if mod == modSet {
		j[key] = data
	} else if mod == modDel {
		delete(j, key)
	}

	w, err := json.MarshalIndent(j, "", "  ")
	if err != nil {
		return err
	}
	return utils.AtomicWriteFile(configFile, w, 0660)
}
