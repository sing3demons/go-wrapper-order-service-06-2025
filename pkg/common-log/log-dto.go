package commonlog

import (
	"fmt"
	"log/slog"
	"os"
)

// ErrorSourceType represents the source of an error.
type ErrorSourceType struct {
	Node        string      `json:"node"`
	Code        interface{} `json:"code"` // string or number
	Description string      `json:"description"`
}

type SequenceResult struct {
	Result  string `json:"Result"`
	Desc    string `json:"Desc"`
	ResTime int64  `json:"ResTime,omitempty"` // Response time in milliseconds
}
type Sequence struct {
	Node    string           `json:"Node"`
	Command string           `json:"Command"`
	Result  []SequenceResult `json:"Result"`
}

type LogEventTag struct {
	Node        string
	Command     string
	Code        string
	Description string
	ResTime     int64
}

// LogDto represents a Data Transfer Object (DTO) for logging information.
type LogDto struct {
	AppName              string      `json:"appName,omitempty"`
	ComponentVersion     string      `json:"componentVersion,omitempty"`
	ComponentName        string      `json:"componentName,omitempty"`
	LogType              string      `json:"logType,omitempty"` // e.g., "application", "system", "error"
	Level                string      `json:"level,omitempty"`   // e.g., "info", "debug", "error"
	Broker               string      `json:"broker,omitempty"`
	Channel              string      `json:"channel,omitempty"`
	UseCase              string      `json:"useCase,omitempty"`
	UseCaseStep          string      `json:"useCaseStep,omitempty"`
	Device               interface{} `json:"device,omitempty"` // string or []string
	Public               string      `json:"public,omitempty"`
	User                 string      `json:"user,omitempty"`
	Action               string      `json:"action,omitempty"`
	SubAction            string      `json:"subAction,omitempty"`
	ActionDescription    string      `json:"actionDescription,omitempty"`
	Message              string      `json:"message,omitempty"`
	Messages             string      `json:"messages,omitempty"`
	Timestamp            string      `json:"timestamp,omitempty"`
	Dependency           string      `json:"dependency,omitempty"`
	ResponseTime         int64       `json:"responseTime,omitempty"`
	ResultCode           string      `json:"resultCode,omitempty"`
	ResultFlag           string      `json:"resultFlag,omitempty"`
	Instance             string      `json:"instance,omitempty"`
	OriginateServiceName string      `json:"originateServiceName,omitempty"`
	RecordName           string      `json:"recordName,omitempty"`
	RecordType           string      `json:"recordType,omitempty"`
	SessionId            string      `json:"sessionId,omitempty"`
	TransactionId        string      `json:"transactionId,omitempty"`
	RequestId            string      `json:"requestId,omitempty"`
	AdditionalInfo       any         `json:"additionalInfo,omitempty"`
	AppResult            string      `json:"appResult,omitempty"`
	AppResultCode        string      `json:"appResultCode,omitempty"`
	DateTime             string      `json:"dateTime,omitempty"`
	ServiceTime          int64       `json:"serviceTime,omitempty"`
	AppResultHttpStatus  string      `json:"appResultHttpStatus,omitempty"`
	AppResultType        string      `json:"appResultType,omitempty"`
	Severity             string      `json:"severity,omitempty"`
	Sequences            []Sequence  `json:"-"` // List of event tags
}

// type ISummaryLogService interface {
// 	Info(msg string)
// 	Error(args ...any)
// }

type LoggerService interface {
	Debugf(format string, args ...any)
	Debug(args ...any)
	Logf(format string, args ...any)
	Log(data string)
	Info(msg string)
	Errorf(format string, args ...any)
	Error(args ...any)
	Sync() error
}

// defaultLoggerService is a default implementation of LoggerService using slog
type defaultLoggerService struct {
	logger *slog.Logger
}

func NewDefaultLoggerService() LoggerService {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{}))

	return &defaultLoggerService{logger: logger}
}

func (l *defaultLoggerService) Debugf(format string, args ...any) {
	l.logger.Debug(fmt.Sprintf(format, args...))
}

func (l *defaultLoggerService) Debug(args ...any) {
	l.logger.Debug(fmt.Sprint(args...))
}

func (l *defaultLoggerService) Logf(format string, args ...any) {
	l.logger.Info(fmt.Sprintf(format, args...)) // treat as general info
}

func (l *defaultLoggerService) Log(data string) {
	l.logger.Info(data)
}

func (l *defaultLoggerService) Info(msg string) {
	l.logger.Info(msg)
}

func (l *defaultLoggerService) Errorf(format string, args ...any) {
	l.logger.Error(fmt.Sprintf(format, args...))
}

func (l *defaultLoggerService) Error(args ...any) {
	l.logger.Error(fmt.Sprint(args...))
}

func (l *defaultLoggerService) Sync() error {
	// slog doesn't require Sync like zap; return nil
	return nil
}

// SummaryParamsType defines summary log parameters.
type SummaryParamsType struct {
	AppResult           string
	AppResultCode       string
	AppResultHttpStatus string
	AppResultType       string
	Severity            string
}

// LogDependencyMetadata defines dependency metadata for logs.
type LogDependencyMetadata struct {
	Dependency   string
	ResponseTime int64
	ResultCode   string
	ResultFlag   string
}
