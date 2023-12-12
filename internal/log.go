package internal

import (
	"github.com/spf13/viper"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/config"
	"io"
	"log/slog"
	"os"
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
	logEnabled := viper.GetBool(config.KeyLog)

	logLevel := viper.GetString(config.KeyLogLevel)
	var level slog.Level
	//levelP := &level
	err := level.UnmarshalText([]byte(logLevel))
	if err != nil {
		level = slog.LevelInfo
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
