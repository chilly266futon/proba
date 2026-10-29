package config

import (
	"strings"
	"testing"
	"time"
)

func setEnv(t *testing.T, kv map[string]string) {
	t.Helper()

	for k, v := range kv {
		t.Setenv(k, v)
	}
}

func TestLoadDevDefaults(t *testing.T) {
	setEnv(t, map[string]string{"APP_ENV": "dev"})

	got, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got.Env != EnvDev {
		t.Errorf("Env = %q, want %q", got.Env, EnvDev)
	}
	if got.HTTP.Addr != ":8080" {
		t.Errorf("HTTP.Addr = %q, want :8080", got.HTTP.Addr)
	}
	if got.JWT.AccessTTL != 15*time.Minute {
		t.Errorf("JWT.AccessTTL = %v, want 15m", got.JWT.AccessTTL)
	}
}

func TestLoadDurationSeconds(t *testing.T) {
	setEnv(t, map[string]string{
		"APP_ENV":                 "dev",
		"JWT_ACCESS_TTL_SECONDS":  "60",
		"JWT_REFRESH_TTL_SECONDS": "3600",
	})

	got, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got.JWT.AccessTTL != time.Minute {
		t.Errorf("AccessTTL = %v, want 1m", got.JWT.AccessTTL)
	}
	if got.JWT.RefreshTTL != time.Hour {
		t.Errorf("RefreshTTL = %v, want 1h", got.JWT.RefreshTTL)
	}
}

func TestLoadProdRequiresMandatory(t *testing.T) {
	setEnv(t, map[string]string{
		"APP_ENV":      "prod",
		"JWT_SECRET":   "s3cret",
		"YDB_ENDPOINT": "grpcs://ydb.example:2135",
		"YDB_DATABASE": "/ru-central1/b1g/test",
		"S3_BUCKET":    "proba-uploads",
	})

	if _, err := Load(); err != nil {
		t.Fatalf("prod with all mandatory: Load() error = %v", err)
	}

	setEnv(t, map[string]string{
		"APP_ENV":      "prod",
		"JWT_SECRET":   "s3cret",
		"YDB_ENDPOINT": "",
		"YDB_DATABASE": "",
		"S3_BUCKET":    "",
	})

	_, err := Load()
	if err == nil {
		t.Fatal("prod without YDB: Load() = nil, want error")
	}
	if !strings.Contains(err.Error(), "YDB_ENDPOINT") || !strings.Contains(err.Error(), "YDB_DATABASE") {
		t.Errorf("error = %q, want mentions of missing vars", err)
	}
}

func TestLoadInvalidEnv(t *testing.T) {
	setEnv(t, map[string]string{"APP_ENV": "staging"})

	if _, err := Load(); err == nil {
		t.Fatal("APP_ENV=staging: Load() = nil, want error")
	}
}
