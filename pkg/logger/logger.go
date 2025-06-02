package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

type ILogger interface {
	Debugf(format string, args ...any)
	Debug(args ...any)
	Logf(format string, args ...any)
	Log(data string)
	Info(msg string)
	Errorf(format string, args ...any)
	Error(args ...any)
	Sync() error
	SummaryLog() *zap.Logger
}

type zLogger struct {
	*zap.Logger
	summaryLogger *zap.Logger
}

func (k *zLogger) Debugf(format string, args ...any) {
	k.Logger.Sugar().Debugf(format, args...)
}

func (k *zLogger) Debug(args ...any) {
	k.Logger.Sugar().Debug(args...)
}

func (k *zLogger) Info(msg string) {
	k.Logger.Info(msg)
}

func (k *zLogger) Logf(format string, args ...any) {
	k.Logger.Sugar().Infof(format, args...)
}
func (k *zLogger) Log(data string) {
	k.Logger.Info(data)
}
func (k *zLogger) Errorf(format string, args ...any) {
	// fmt.Printf(format, args...)
	k.Logger.Sugar().Errorf(format, args...)
}
func (k *zLogger) Error(args ...any) {
	k.Logger.Sugar().Error(args...)
}

func NewLogger() ILogger {
	encoderConfig := map[string]any{
		"messageKey": "msg",
		"InitialFields": map[string]any{
			"pid": os.Getpid(),
		},
	}
	data, _ := json.Marshal(encoderConfig)
	var encCfg zapcore.EncoderConfig
	if err := json.Unmarshal(data, &encCfg); err != nil {
		return nil
	}

	// add the encoder config and rotator to create a new zap logger
	w := zapcore.AddSync(os.Stdout)
	encoder := zapcore.NewConsoleEncoder(encCfg)
	core := zapcore.NewCore(
		encoder,
		w,
		zap.InfoLevel)

	summaryCore := zapcore.NewCore(
		encoder,
		w,
		zap.InfoLevel)

	if os.Getenv("LOG_PATH") != "" {
		path := os.Getenv("LOG_PATH")
		logFile := filepath.Join(path, getLogFileName(time.Now()))
		if _, err := os.Stat(path); os.IsNotExist(err) {
			if err := os.MkdirAll(path, 0755); err != nil {
				fmt.Printf("Failed to create log directory: %v\n", err)
				return nil
			}
		}
		fileEncoder := zapcore.NewConsoleEncoder(encCfg)
		// Setting up lumberjack logger for log rotation
		writerSync := zapcore.AddSync(&lumberjack.Logger{
			Filename:   logFile,
			MaxSize:    500, // megabytes
			MaxBackups: 3,   // number of backups
			MaxAge:     1,   // days
			LocalTime:  true,
			Compress:   true, // compress the backups
		})

		logSummaryFile := filepath.Join("logs/summary", getLogFileName(time.Now()))
		if _, err := os.Stat(filepath.Join("logs", "summary")); os.IsNotExist(err) {
			if err := os.MkdirAll(filepath.Join("logs", "summary"), 0755); err != nil {
				fmt.Printf("Failed to create summary log directory: %v\n", err)
				return nil
			}
		}

		writerSyncSummary := zapcore.AddSync(&lumberjack.Logger{
			Filename:   logSummaryFile,
			MaxSize:    500, // megabytes
			MaxBackups: 3,   // number of backups
			MaxAge:     1,   // days
			LocalTime:  true,
			Compress:   true, // compress the backups
		})

		core = zapcore.NewTee(
			zapcore.NewCore(fileEncoder, writerSync, zap.InfoLevel),
			zapcore.NewCore(zapcore.NewConsoleEncoder(encCfg), w, zap.InfoLevel),
		)

		summaryCore = zapcore.NewTee(
			zapcore.NewCore(fileEncoder, writerSyncSummary, zap.InfoLevel),
			zapcore.NewCore(zapcore.NewConsoleEncoder(encCfg), w, zap.InfoLevel),
		)
	}

	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zap.ErrorLevel))

	if os.Getenv("MODE") == "test" {
		logger = zap.NewNop()
	}
	customLog := &zLogger{Logger: logger,
		summaryLogger: zap.New(summaryCore, zap.AddCaller(), zap.AddStacktrace(zap.ErrorLevel))}

	return customLog
}

func (k *zLogger) Sync() error {
	return k.Logger.Sync()
}

func (k *zLogger) SummaryLog() *zap.Logger {
	return k.summaryLogger
}

func getLogFileName(t time.Time) string {
	appName := ""
	year, month, day := t.Date()
	hour, minute, second := t.Clock()

	return fmt.Sprintf("%s_%04d%02d%02d_%02d%02d%02d.log", appName, year, month, day, hour, minute, second)
}
