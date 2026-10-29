// Package http — транспортный слой платформы: роутер, middleware, ответы.
//
// Формат ошибок соответствует контракту: Error {code, message, details, request_id}.
package http

import (
	stdhttp "net/http"
	"time"

	"github.com/rs/zerolog"
)

// ServerOpts — параметры сервера.
type ServerOpts struct {
	// Log — корневой логгер; если не задан, логи отключаются (zerolog.Nop).
	Log zerolog.Logger
}

// Server собирает типовой HTTP-обработчик с middleware-цепочкой.
type Server struct {
	log   zerolog.Logger
	mux   *stdhttp.ServeMux
	chain []func(stdhttp.Handler) stdhttp.Handler
}

// New создаёт сервер с базовой цепочкой middleware: request_id → log → recover.
// Recover — самый внутренний, чтобы запрос с паникой всё равно попал в access-лог со статусом 500.
func New(opts ServerOpts) *Server {
	log := opts.Log
	if log.GetLevel() == zerolog.NoLevel || log.GetLevel() == zerolog.Disabled {
		log = zerolog.Nop()
	}

	s := &Server{
		log:   log,
		mux:   stdhttp.NewServeMux(),
		chain: []func(stdhttp.Handler) stdhttp.Handler{},
	}
	s.Use(RequestID)
	s.Use(Logging)
	s.Use(Recover)

	return s
}

// Use добавляет middleware в конец цепочки: первый добавленный — самый внешний,
// последний — ближе всех к обработчику.
func (s *Server) Use(mw func(stdhttp.Handler) stdhttp.Handler) {
	s.chain = append(s.chain, mw)
}

// Handle регистрирует обработчик по методу и пути (паттерны ServeMux Go 1.22+).
func (s *Server) Handle(method, path string, h stdhttp.HandlerFunc) {
	s.mux.Handle(method+" "+path, h)
}

// Handler возвращает финальный обработчик.
func (s *Server) Handler() stdhttp.Handler {
	var h stdhttp.Handler = s.mux
	for i := len(s.chain) - 1; i >= 0; i-- {
		h = s.chain[i](h)
	}

	// Корневой логгер прокидывается в контекст самым внешним слоем,
	// чтобы middleware Logging видел его для всех запросов.
	return stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		h.ServeHTTP(w, r.WithContext(WithLogger(r.Context(), s.log)))
	})
}

// ShutdownTimeout — таймаут корректной остановки (используется в cmd/api).
const ShutdownTimeout = 10 * time.Second
