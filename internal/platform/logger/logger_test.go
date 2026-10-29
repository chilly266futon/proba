package logger

import (
	"testing"

	"github.com/rs/zerolog"
)

func TestNewRespectsLevel(t *testing.T) {
	log := New("debug", false)
	if log.GetLevel() != zerolog.DebugLevel {
		t.Errorf("level = %s, want debug", log.GetLevel())
	}
}

func TestNewFallsBackToInfoOnUnknownLevel(t *testing.T) {
	log := New("bogus", false)
	if log.GetLevel() != zerolog.InfoLevel {
		t.Errorf("level = %s, want info", log.GetLevel())
	}
}

func TestNewWrites(t *testing.T) {
	log := New("info", true)
	log.Info().Msg("test message") // не должно упасть в любом режиме
	t.Log("log write ok")
}
