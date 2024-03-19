package cmd

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wot-oss/tmc/internal/config"
)

func buildTMCEnvVar(name string) string {
	return strings.ToUpper(config.EnvPrefix + "_" + name)
}

func TestGetServerOptionsReadsFromEnvironment(t *testing.T) {

	t.Run("with set environment variables", func(t *testing.T) {
		envAllowedHeaders := buildTMCEnvVar(config.KeyCorsAllowedHeaders)
		envAllowedOrigins := buildTMCEnvVar(config.KeyCorsAllowedOrigins)
		envAllowCredentials := buildTMCEnvVar(config.KeyCorsAllowCredentials)
		envMaxAge := buildTMCEnvVar(config.KeyCorsMaxAge)

		t.Setenv(envAllowedHeaders, "X-Api-Key, X-Bar")
		t.Setenv(envAllowedOrigins, "http://example.org, https://sample.com")
		t.Setenv(envAllowCredentials, "true")
		t.Setenv(envMaxAge, "120")
		config.InitViper()

		opts := getCORSOptions()

		corsOrigins := fmt.Sprintf("%v", reflect.ValueOf(opts).FieldByName("allowedOrigins"))
		corsHeaders := fmt.Sprintf("%v", reflect.ValueOf(opts).FieldByName("allowedHeaders"))
		corsCredentials := fmt.Sprintf("%v", reflect.ValueOf(opts).FieldByName("allowCredentials"))
		corsMaxAge := fmt.Sprintf("%v", reflect.ValueOf(opts).FieldByName("maxAge"))

		assert.True(t, strings.Contains(corsOrigins, "http://example.org"))
		assert.True(t, strings.Contains(corsOrigins, "https://sample.com"))
		assert.True(t, strings.Contains(corsHeaders, "X-Api-Key"))
		assert.True(t, strings.Contains(corsHeaders, "X-Bar"))
		assert.Equal(t, "true", corsCredentials)
		assert.Equal(t, "120", corsMaxAge)
	})

	t.Run("without set environment variables", func(t *testing.T) {
		config.InitViper()

		opts := getCORSOptions()

		corsOrigins := fmt.Sprintf("%v", reflect.ValueOf(opts).FieldByName("allowedOrigins"))
		corsHeaders := fmt.Sprintf("%v", reflect.ValueOf(opts).FieldByName("allowedHeaders"))
		corsCredentials := fmt.Sprintf("%v", reflect.ValueOf(opts).FieldByName("allowCredentials"))
		corsMaxAge := fmt.Sprintf("%v", reflect.ValueOf(opts).FieldByName("maxAge"))

		assert.Equal(t, "[]", corsOrigins)
		assert.Equal(t, "[]", corsHeaders)
		assert.Equal(t, "false", corsCredentials)
		assert.Equal(t, "0", corsMaxAge)
	})
}
