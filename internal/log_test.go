package internal

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/config"
)

const envVarLog = config.EnvPrefix + "_" + config.KeyLog

func TestLogDisabledDefault(t *testing.T) {
	// given: default environment with no env var set that would enable logging
	t.Setenv(envVarLog, "")
	config.InitViper()

	// when: initialize the logging
	InitLogging()
	hdl := slog.Default().Handler()
	_, isDiscardHandler := hdl.(*DiscardLogHandler)

	// then: the logs are written to the discard handler
	assert.True(t, isDiscardHandler)
}

func TestLogEnabledByEnvVar(t *testing.T) {
	// given: environment with env var set that enables logging
	t.Setenv(envVarLog, "true")
	config.InitViper()

	// when: initialize the logging
	InitLogging()
	hdl := slog.Default().Handler()
	_, isDefaultHandler := hdl.(*DefaultLogHandler)

	// then: the logs are written to the default handler
	assert.True(t, isDefaultHandler)

	t.Setenv(envVarLog, "")
}
