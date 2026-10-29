// Package logger — единая точка настройки zerolog.
//
// В dev используется человекочитаемый консольный вывод, в prod — JSON.
// Запрещено использовать стандартные log/log/slog (см. depguard в .golangci.yml).
package logger

import (
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
)

// New собирает главный логгер процесса.
//
// level принимает значения zerolog: trace, debug, info, warn, error, fatal, panic.
// pretty=true включает консольный вывод (используется в dev-окружении).
func New(level string, pretty bool) zerolog.Logger {
	lvl, err := zerolog.ParseLevel(level)
	if err != nil { // нераспознанный уровень -> info
		lvl = zerolog.InfoLevel
	}

	var out io.Writer = os.Stdout
	if pretty {
		out = zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	}

	zerolog.TimeFieldFormat = time.RFC3339

	return zerolog.New(out).Level(lvl).With().Timestamp().Logger()
}
