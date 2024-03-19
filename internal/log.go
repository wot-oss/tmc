package internal

import (
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/viper"
	"github.com/wot-oss/tmc/internal/config"
)

type DefaultLogHandler struct {
	*slog.TextHandler
}

type DiscardLogHandler struct {
	*slog.TextHandler
}

func newDefaultLogHandler(opts *slog.HandlerOptions) slog.Handler {
	return &DefaultLogHandler{
		TextHandler: slog.NewTextHandler(os.Stderr, opts),
	}
}

func newDiscardLogHandler(opts *slog.HandlerOptions) slog.Handler {
	return &DiscardLogHandler{
		TextHandler: slog.NewTextHandler(io.Discard, opts),
	}
}

func InitLogging() {
	logLevel := viper.GetString(config.KeyLogLevel)

	var logEnabled bool
	level := slog.LevelError

	switch logLevel {
	case "":
		logEnabled = false
	case strings.ToLower(config.LogLevelOff):
		logEnabled = false
	default:
		logEnabled = true
		err := level.UnmarshalText([]byte(logLevel))
		if err != nil {
			level = slog.LevelInfo
		}
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	var handler slog.Handler
	if logEnabled {
		handler = newDefaultLogHandler(opts)
	} else {
		handler = newDiscardLogHandler(opts)
	}

	log := slog.New(handler)
	slog.SetDefault(log)
}
