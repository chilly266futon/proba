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
	srv.mux.HandleFunc("GET /x", func(_ stdhttp.ResponseWriter, r *stdhttp.Request) {
		got = RequestIDFrom(r.Context())
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
	srv.mux.HandleFunc("GET /x", func(stdhttp.ResponseWriter, *stdhttp.Request) {})

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
	srv.mux.HandleFunc("GET /x", func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		WriteNoContent(w)
	})

	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest("GET", "/x", nil))

	if rec.Code != stdhttp.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}

	lines := strings.Split(strings.TrimSpace(sb.String()), "\n")
	if len(lines) != 1 {
		t.Fatalf("access-лог: %d записей, want 1: %q", len(lines), sb.String())
	}

	var entry struct {
		Status int    `json:"status"`
		Method string `json:"method"`
		Path   string `json:"path"`
	}
	if err := json.Unmarshal([]byte(lines[0]), &entry); err != nil {
		t.Fatalf("access-лог не JSON: %v", err)
	}
	if entry.Status != 204 || entry.Method != "GET" || entry.Path != "/x" {
		t.Errorf("access-лог = %+v, want GET /x 204", entry)
	}
}

func TestRequestIDRejectsUnsafeIncoming(t *testing.T) {
	for _, bad := range []string{"has space", "new\nline", strings.Repeat("a", maxRequestIDLen+1), "кириллица"} {
		srv := New(ServerOpts{})
		srv.mux.HandleFunc("GET /x", func(stdhttp.ResponseWriter, *stdhttp.Request) {})

		req := httptest.NewRequest("GET", "/x", nil)
		req.Header[HeaderRequestID] = []string{bad} // в обход канонизации и проверок httptest
		rec := httptest.NewRecorder()

		srv.Handler().ServeHTTP(rec, req)

		if got := rec.Header().Get(HeaderRequestID); got == bad || !validRequestID(got) {
			t.Errorf("входящий %q: ответный id = %q, want новый валидный", bad, got)
		}
	}
}

func TestPanicIsLoggedWith500(t *testing.T) {
	var sb strings.Builder
	srv := New(ServerOpts{Log: zerolog.New(&sb).Level(zerolog.InfoLevel)})
	srv.mux.HandleFunc("GET /panic", func(stdhttp.ResponseWriter, *stdhttp.Request) { panic("boom") })

	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest("GET", "/panic", nil))

	if !strings.Contains(sb.String(), `"status":500`) {
		t.Errorf("запрос с паникой не попал в access-лог со статусом 500: %q", sb.String())
	}
}
