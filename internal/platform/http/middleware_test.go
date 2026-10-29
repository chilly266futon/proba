package http

import (
	"encoding/json"
	stdhttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rs/zerolog"
)

func TestRequestIDGeneratesAndPropagates(t *testing.T) {
	var got string

	srv := New(ServerOpts{})
	srv.Use(RequestID)
	srv.mux.HandleFunc("GET /x", func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		got = RequestIDFrom(r.Context())
		w.Header().Set(HeaderRequestID, got)
	})

	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest("GET", "/x", nil))

	if got == "" {
		t.Fatal("request_id не сгенерирован")
	}
	if rec.Header().Get(HeaderRequestID) != got {
		t.Errorf("ответный X-Request-Id = %q, want %q", rec.Header().Get(HeaderRequestID), got)
	}
}

func TestRequestIDReusesIncoming(t *testing.T) {
	incoming := "req-12345"

	srv := New(ServerOpts{})
	srv.Use(RequestID)
	srv.mux.HandleFunc("GET /x", func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		w.Header().Set(HeaderRequestID, RequestIDFrom(r.Context()))
	})

	req := httptest.NewRequest("GET", "/x", nil)
	req.Header.Set(HeaderRequestID, incoming)
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Header().Get(HeaderRequestID) != incoming {
		t.Errorf("X-Request-Id = %q, want %q", rec.Header().Get(HeaderRequestID), incoming)
	}
}

func TestRecoverAnswersContractError(t *testing.T) {
	srv := New(ServerOpts{})
	srv.Use(RequestID)
	srv.Use(Recover)
	srv.mux.HandleFunc("GET /panic", func(stdhttp.ResponseWriter, *stdhttp.Request) {
		panic("boom")
	})

	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest("GET", "/panic", nil))

	if rec.Code != stdhttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}

	var body ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body.Code != CodeInternal {
		t.Errorf("code = %q, want internal_error", body.Code)
	}
	if body.RequestID == "" {
		t.Error("request_id отсутствует в ответе ошибки")
	}
}

func TestLoggingWritesAccessLog(t *testing.T) {
	var sb strings.Builder
	log := zerolog.New(&sb).Level(zerolog.InfoLevel)

	srv := New(ServerOpts{Log: log})
	srv.Use(RequestID)
	srv.Use(Logging)
	srv.mux.HandleFunc("GET /x", func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		WriteNoContent(w)
	})

	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest("GET", "/x", nil))

	if rec.Code != stdhttp.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
	if !strings.Contains(sb.String(), "status=204") {
		t.Errorf("access-лог не содержит статус 204: %q", sb.String())
	}
}
