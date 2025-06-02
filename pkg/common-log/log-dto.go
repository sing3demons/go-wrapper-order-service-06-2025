package commonlog

import "go.uber.org/zap"

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

type EventTag struct {
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
	AdditionalInfo       any         `json:"additionalInfo,omitempty"`
	AppResult            string      `json:"appResult,omitempty"`
	AppResultCode        string      `json:"appResultCode,omitempty"`
	DateTime             string      `json:"dateTime,omitempty"`
	ServiceTime          int64       `json:"serviceTime,omitempty"`
	AppResultHttpStatus  string      `json:"appResultHttpStatus,omitempty"`
	AppResultType        string      `json:"appResultType,omitempty"`
	Severity             string      `json:"severity,omitempty"`
	Sequences            []Sequence  `json:"sequences,omitempty"` // List of event tags
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

	SummaryLog() *zap.Logger
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
