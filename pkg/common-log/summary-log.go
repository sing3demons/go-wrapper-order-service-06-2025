package commonlog

import (
	"encoding/json"
	"reflect"
	"time"

	"github.com/sing3demons/go-order-service/pkg/common-log/masking"
	"go.uber.org/zap"
)

type SummaryLogService interface {
	Init(data LogDto)
	Update(key string, value any)
	Flush()
}
type summaryLogService struct {
	logDto         LogDto
	logger         *zap.Logger
	maskingService masking.MaskingService
	customLogger   *customLoggerService
}

func NewSummaryLogService(logger *zap.Logger, customLogger *customLoggerService) SummaryLogService {
	return &summaryLogService{
		logger:         logger,
		maskingService: *maskingService,
		customLogger:   customLogger,
		logDto:         customLogger.logDto,
	}
}

func (c *summaryLogService) clearNonSummaryLogParam() {
	c.logDto.Action = ""
	c.logDto.Message = ""
	c.logDto.Timestamp = ""
	c.logDto.Dependency = ""
	c.logDto.ResponseTime = 0
	c.logDto.ResultCode = ""
	c.logDto.ResultFlag = ""
	c.logDto.ActionDescription = ""
}

func (s *summaryLogService) Init(data LogDto) {
	s.logDto = data
	s.logDto.LogType = "Summary"
}

func (s *summaryLogService) Update(key string, value any) {
	v := reflect.ValueOf(&s.logDto).Elem()
	field := v.FieldByName(key)
	if field.IsValid() && field.CanSet() {
		field.Set(reflect.ValueOf(value))
	}
}

func (s *summaryLogService) Flush() {
	s.Init(s.customLogger.logDto)
	s.logDto.RecordType = "Summary"
	s.logDto.DateTime = s.customLogger.logDto.DateTime
	startTime, err := time.Parse(time.RFC3339, s.customLogger.logDto.DateTime)
	if err == nil {
		s.logDto.ServiceTime = time.Since(startTime).Milliseconds() / 1000
	} else {
		s.logDto.ServiceTime = 0
	}

	if s.customLogger.additionalSummary != nil {
		s.logDto.AdditionalInfo = s.customLogger.additionalSummary
		s.customLogger.additionalSummary = nil
	}

	if s.customLogger.logDto.AppResultHttpStatus != "" {
		s.logDto.AppResultHttpStatus = s.customLogger.logDto.AppResultHttpStatus
	} else {
		s.logDto.AppResultHttpStatus = "200"
	}

	if s.customLogger.logDto.AppResultType != "" {
		s.logDto.AppResultType = s.customLogger.logDto.AppResultType
	} else {
		s.logDto.AppResultType = "HEALTHY"
	}

	if s.customLogger.logDto.Severity != "" {
		s.logDto.Severity = s.customLogger.logDto.Severity
	} else {
		s.logDto.Severity = "NOTICE"
	}

	if s.customLogger.logDto.AppResult != "" {
		s.logDto.AppResult = s.customLogger.logDto.AppResult
	} else {
		s.logDto.AppResult = "Success"
	}

	if s.customLogger.logDto.AppResultCode != "" {
		s.logDto.AppResultCode = s.customLogger.logDto.AppResultCode
	} else {
		s.logDto.AppResultCode = "20000"
	}

	if len(s.customLogger.summaryLogAdditionalInfo) > 0 {
		s.logDto.Sequences = append(s.logDto.Sequences, s.customLogger.summaryLogAdditionalInfo...)
		s.customLogger.summaryLogAdditionalInfo = nil
	}

	s.clearNonSummaryLogParam()
	jsonBytes, err := json.Marshal(s.logDto)
	if err != nil {
		s.logger.Error("Failed to marshal summary log data", zap.Error(err))
		return

	}
	info := string(jsonBytes)
	s.logger.Info(info)
}

func (s *summaryLogService) FlushError(data any) {
	s.logDto.RecordType = "Summary"
	s.logDto.DateTime = s.customLogger.logDto.DateTime
	startTime, err := time.Parse(time.RFC3339, s.customLogger.logDto.DateTime)
	if err == nil {
		s.logDto.ServiceTime = time.Since(startTime).Milliseconds() / 1000
	} else {
		s.logDto.ServiceTime = 0
	}

	if s.customLogger.additionalSummary != nil {
		s.logDto.AdditionalInfo = s.customLogger.additionalSummary
		s.customLogger.additionalSummary = nil
	}

	s.clearNonSummaryLogParam()
	s.logDto.Message = toJSON(data)
	jsonBytes, err := json.Marshal(s.logDto)
	if err != nil {
		s.logger.Error("Failed to marshal summary log data", zap.Error(err))
		return
	}
	info := string(jsonBytes)
	s.logger.Error(info)
}
