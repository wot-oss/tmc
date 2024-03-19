package internal

import (
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wot-oss/tmc/internal/config"
)

var envVarLogLevel = strings.ToUpper(config.EnvPrefix + "_" + config.KeyLogLevel) // TMC_LOG

func TestLogDisabledByLogLevelEmpty(t *testing.T) {
	// given: default environment with no environment variable set for loglevel
	t.Setenv(envVarLogLevel, "")
	config.InitViper()

	// when: initialize the logging
	InitLogging()
	hdl := slog.Default().Handler()
	_, isDiscardHandler := hdl.(*DiscardLogHandler)

	// then: the logs are written to the discard handler
	assert.True(t, isDiscardHandler)
}

func TestLogDisabledByLogLevelOff(t *testing.T) {
	// given: default environment with environment variable for loglevel is set to "off"
	t.Setenv(envVarLogLevel, "off")
	config.InitViper()

	// when: initialize the logging
	InitLogging()
	hdl := slog.Default().Handler()
	_, isDiscardHandler := hdl.(*DiscardLogHandler)

	// then: the logs are written to the discard handler
	assert.True(t, isDiscardHandler)
}

func TestLogEnabledByLogLevel(t *testing.T) {
	tests := []struct {
		in  string
		out slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"DEBUG", slog.LevelDebug},
		{"dEbuG", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"error", slog.LevelError},
		{"if not known set default info", slog.LevelInfo},
	}

	for _, test := range tests {
		// when: setting loglevel via environment variable
		t.Setenv(envVarLogLevel, test.in)
		// and when: initialize the logging
		config.InitViper()
		InitLogging()

		hdl := slog.Default().Handler()
		lh, isDefaultHandler := hdl.(*DefaultLogHandler)

		// then: the logs are written to the default handler
		assert.True(t, isDefaultHandler)
		// and then: the expected loglevel is enabled
		assert.True(t, lh.Enabled(nil, test.out))
	}
}
