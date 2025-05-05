package logger_test

import (
	"os"
	"testing"

	"github.com/sing3demons/go-order-service/pkg/logger"
)

func TestNewLogger(t *testing.T) {
	os.Setenv("MODE", "test")
	defer os.Unsetenv("MODE")
	log := logger.NewLogger()

	// These won't panic but print to stderr (zap)
	log.Debug("debug message")
	log.Debugf("debugf: %d", 1)
	log.Log("log message")
	log.Logf("logf: %s", "info")
	log.Error("error message")
	log.Errorf("errorf: %v", "error")

	if err := log.Sync(); err != nil {
		t.Errorf("expected no sync error, got: %v", err)
	}
}
