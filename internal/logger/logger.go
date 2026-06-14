package logger

import (
	"log/slog"
	"os"
)

type Options struct {
	AppName     string
	Environment string
}

func New(opts Options) *slog.Logger {
	var logger *slog.Logger

	switch opts.Environment {
	case "local", "dev":
		logger = slog.New(NewPrettyLogHandler(os.Stdout, slog.LevelDebug))
	case "prod":
	default:
		logger = slog.New(
			slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	}

	if opts.AppName != "" {
		logger = logger.With(
			slog.String("app", opts.AppName),
		)
	}

	return logger
}

// Error returns a [slog.Attr] that represents error.
//
// In perfect world every log message that contains error output should be built with this helper.
func Error(err error) slog.Attr {
	if err == nil {
		return slog.Any("error", err)
	}

	return slog.String("error", err.Error())
}

// WithComponent returns a [*slog.Logger] that includes component attribute with the given name in each
// subsequent output operation.
func WithComponent(logger *slog.Logger, name string) *slog.Logger {
	return logger.With(
		slog.String("component", name),
	)
}
