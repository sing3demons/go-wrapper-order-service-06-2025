package logger

import (
	"os"

	"go.uber.org/zap"
)

type ILogger interface {
	Debugf(format string, args ...any)
	Debug(args ...any)
	Logf(format string, args ...any)
	Log(args ...any)
	Errorf(format string, args ...any)
	Error(args ...any)
	Sync() error
}

type zLogger struct {
	*zap.Logger
}

func (k *zLogger) Debugf(format string, args ...any) {
	k.Logger.Sugar().Debugf(format, args...)
}

func (k *zLogger) Debug(args ...any) {
	k.Logger.Sugar().Debug(args...)
}

func (k *zLogger) Logf(format string, args ...any) {
	k.Logger.Sugar().Infof(format, args...)
}
func (k *zLogger) Log(args ...any) {
	k.Logger.Sugar().Info(args...)
}
func (k *zLogger) Errorf(format string, args ...any) {
	// fmt.Printf(format, args...)
	k.Logger.Sugar().Errorf(format, args...)
}
func (k *zLogger) Error(args ...any) {
	k.Logger.Sugar().Error(args...)
}

func NewLogger() ILogger {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	if os.Getenv("MODE") == "test" {
		logger = zap.NewNop()
	}
	return &zLogger{Logger: logger}
}

func (k *zLogger) Sync() error {
	return k.Logger.Sync()
}
