package http

import (
	"context"
	"net"
	stdhttp "net/http"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// HeaderRequestID — заголовок с идентификатором запроса.
const HeaderRequestID = "X-Request-Id"

type (
	requestIDCtxKey struct{}
	loggerCtxKey    struct{}
)

// RequestID генерирует (или прокидывает) request_id на каждый запрос.
func RequestID(next stdhttp.Handler) stdhttp.Handler {
	return stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		id := r.Header.Get(HeaderRequestID)
		if id == "" {
			id = uuid.NewString()
		}

		ctx := context.WithValue(r.Context(), requestIDCtxKey{}, id)
		r = r.WithContext(ctx)

		next.ServeHTTP(&statusWriter{ResponseWriter: w, code: stdhttp.StatusOK, id: id}, r)
	})
}

// Recover перехватывает панику и отвечает 500 по контракту.
func Recover(next stdhttp.Handler) stdhttp.Handler {
	return stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				l := loggerFor(r)
				l.Error().Any("panic", rec).Msg("http panic recovered")
				WriteError(w, stdhttp.StatusInternalServerError, CodeInternal, "внутренняя ошибка", nil)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// Logging пишет access-лог по каждому запросу.
func Logging(next stdhttp.Handler) stdhttp.Handler {
	return stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)

		l := loggerFor(r)
		l.Info().
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Int("status", extractStatus(w)).
			Dur("duration", time.Since(start)).
			Msg("http")
	})
}

// --- вспомогательное ---

// statusWriter запоминает код ответа и request_id.
type statusWriter struct {
	stdhttp.ResponseWriter
	code int
	id   string
}

func (w *statusWriter) WriteHeader(code int) {
	w.code = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Status() int       { return w.code }
func (w *statusWriter) RequestID() string { return w.id }

func extractStatus(w stdhttp.ResponseWriter) int {
	if sw, ok := w.(interface{ Status() int }); ok {
		return sw.Status()
	}

	return stdhttp.StatusOK
}

// WithLogger кладёт логгер в контекст запроса.
func WithLogger(ctx context.Context, log zerolog.Logger) context.Context {
	return context.WithValue(ctx, loggerCtxKey{}, log)
}

func loggerFor(r *stdhttp.Request) zerolog.Logger {
	if l, ok := r.Context().Value(loggerCtxKey{}).(zerolog.Logger); ok {
		return l
	}

	return zerolog.Nop()
}

// RequestIDFrom достаёт request_id из контекста запроса.
func RequestIDFrom(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDCtxKey{}).(string); ok {
		return id
	}

	return ""
}

// RemoteIP достаёт IP клиента (X-Forwarded-For от API Gateway).
func RemoteIP(r *stdhttp.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if idx := indexByte(xff, ','); idx > 0 {
			return xff[:idx]
		}

		return xff
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}

	return host
}

func indexByte(s string, b byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return i
		}
	}

	return -1
}
