package cmd

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/config"
)

func buildTMCEnvVar(name string) string {
	return strings.ToUpper(config.EnvPrefix + "_" + name)
}

func TestGetServerOptionsReadsFromEnvironment(t *testing.T) {

	t.Run("with set environment variables", func(t *testing.T) {
		envAllowedHeaders := buildTMCEnvVar(config.KeyCorsAllowedHeaders)
		envAllowedOrigins := buildTMCEnvVar(config.KeyCorsAllowedOrigins)
		envAllowCredentials := buildTMCEnvVar(config.KeyCorsAllowCredentials)

		t.Setenv(envAllowedHeaders, "X-Api-Key, X-Bar")
		t.Setenv(envAllowedOrigins, "http://example.org, https://sample.com")
		t.Setenv(envAllowCredentials, "true")
		config.InitViper()

		opts := getServerOptions()

		corsOrigins := fmt.Sprintf("%v", reflect.ValueOf(opts.CORS).FieldByName("allowedOrigins"))
		corsHeaders := fmt.Sprintf("%v", reflect.ValueOf(opts.CORS).FieldByName("allowedHeaders"))
		corsCredentials := fmt.Sprintf("%v", reflect.ValueOf(opts.CORS).FieldByName("allowCredentials"))

		assert.True(t, strings.Contains(corsOrigins, "http://example.org"))
		assert.True(t, strings.Contains(corsOrigins, "https://sample.com"))
		assert.True(t, strings.Contains(corsHeaders, "X-Api-Key"))
		assert.True(t, strings.Contains(corsHeaders, "X-Bar"))
		assert.Equal(t, "true", corsCredentials)
	})

	t.Run("without set environment variables", func(t *testing.T) {
		config.InitViper()

		opts := getServerOptions()

		corsOrigins := fmt.Sprintf("%v", reflect.ValueOf(opts.CORS).FieldByName("allowedOrigins"))
		corsHeaders := fmt.Sprintf("%v", reflect.ValueOf(opts.CORS).FieldByName("allowedHeaders"))
		corsCredentials := fmt.Sprintf("%v", reflect.ValueOf(opts.CORS).FieldByName("allowCredentials"))

		assert.Equal(t, "[]", corsOrigins)
		assert.Equal(t, "[]", corsHeaders)
		assert.Equal(t, "false", corsCredentials)
	})
}
