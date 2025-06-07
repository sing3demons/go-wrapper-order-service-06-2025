package commonlog

import (
	"encoding/json"
	"reflect"
	"time"

	"github.com/sing3demons/go-order-service/pkg/common-log/LogSeverity"
	"github.com/sing3demons/go-order-service/pkg/common-log/masking"
)

type SummaryLogService interface {
	Init(data LogDto)
	Update(key string, value any)
	Flush()
	FlushError(data Stack)
	End(data Stack)
}
type summaryLogService struct {
	logDto         LogDto
	logger         LoggerService
	maskingService masking.MaskingService
	customLogger   *customLoggerService
}

func NewSummaryLogService(logger LoggerService, customLogger *customLoggerService) SummaryLogService {
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
	s.logDto.DateTime = time.Unix(s.customLogger.utilService.now, 0).Format(time.RFC3339)
	s.logDto.ServiceTime = time.Since(s.customLogger.utilService.begin).Microseconds()

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
		s.logDto.Severity = LogSeverity.NORMAL
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
		sequences := s.logDto.Sequences
		jsonBytes, err := json.Marshal(sequences)
		if err == nil {
			s.logDto.Messages = string(jsonBytes)
		}
		s.logDto.Sequences = nil
		s.customLogger.summaryLogAdditionalInfo = nil
	}

	s.clearNonSummaryLogParam()
	jsonBytes, err := json.Marshal(s.logDto)
	if err == nil {
		info := string(jsonBytes)
		s.logger.Info(info)
	}
}

type Stack struct {
	Status     string `json:"status,omitempty"`
	ResultType string `json:"resultType,omitempty"`
	Severity   string `json:"severity,omitempty"`
	Message    string `json:"message,omitempty"`
	Code       string `json:"code,omitempty"`
}

func (s *summaryLogService) FlushError(data Stack) {
	s.Init(s.customLogger.logDto)
	s.logDto.RecordType = "Summary"
	s.logDto.DateTime = time.Unix(s.customLogger.utilService.now, 0).Format(time.RFC3339)
	s.logDto.ServiceTime = time.Since(s.customLogger.utilService.begin).Microseconds()

	if s.customLogger.additionalSummary != nil {
		s.logDto.AdditionalInfo = s.customLogger.additionalSummary
		s.customLogger.additionalSummary = nil
	}

	if data.Status != "" {
		s.logDto.AppResultHttpStatus = data.Status
	} else {
		s.logDto.AppResultHttpStatus = "200"
	}

	if data.ResultType != "" {
		s.logDto.AppResultType = data.ResultType
	} else {
		s.logDto.AppResultType = "SYSTEM_ERROR"
	}

	if data.Severity != "" {
		s.logDto.Severity = data.Severity
	} else {
		s.logDto.Severity = LogSeverity.NOTICE
	}

	if data.Message != "" {
		s.logDto.AppResult = data.Message
	}

	if data.Code != "" {
		s.logDto.AppResultCode = data.Code
	} else {
		s.logDto.AppResultCode = "50000"
	}
	if len(s.customLogger.summaryLogAdditionalInfo) > 0 {
		s.logDto.Sequences = append(s.logDto.Sequences, s.customLogger.summaryLogAdditionalInfo...)
		sequences := s.logDto.Sequences
		jsonBytes, err := json.Marshal(sequences)
		if err == nil {
			s.logDto.Messages = string(jsonBytes)
		}
		s.logDto.Sequences = nil
		s.customLogger.summaryLogAdditionalInfo = nil
	}

	s.clearNonSummaryLogParam()
	jsonBytes, err := json.Marshal(s.logDto)
	if err == nil {
		info := string(jsonBytes)
		s.logger.Info(info)
	}
}

func (s *summaryLogService) End(data Stack) {
	s.Init(s.customLogger.logDto)
	s.logDto.RecordType = "Summary"
	s.logDto.DateTime = time.Unix(s.customLogger.utilService.now, 0).Format(time.RFC3339)
	s.logDto.ServiceTime = time.Since(s.customLogger.utilService.begin).Microseconds()
	if s.customLogger.additionalSummary != nil {
		s.logDto.AdditionalInfo = s.customLogger.additionalSummary
		s.customLogger.additionalSummary = nil
	}

	if data.Status != "" {
		s.logDto.AppResultHttpStatus = data.Status
	} else if s.customLogger.logDto.AppResultHttpStatus != "" {
		s.logDto.AppResultHttpStatus = s.customLogger.logDto.AppResultHttpStatus
	} else {
		s.logDto.AppResultHttpStatus = "200"
	}

	if data.ResultType != "" {
		s.logDto.AppResultType = data.ResultType
	} else if s.customLogger.logDto.AppResultType != "" {
		s.logDto.AppResultType = s.customLogger.logDto.AppResultType
	} else {
		s.logDto.AppResultType = "HEALTHY"
	}

	if data.Severity != "" {
		s.logDto.Severity = data.Severity
	} else if s.customLogger.logDto.Severity != "" {
		s.logDto.Severity = s.customLogger.logDto.Severity
	} else {
		s.logDto.Severity = LogSeverity.NORMAL
	}

	if data.Message != "" {
		s.logDto.AppResult = data.Message
	} else if s.customLogger.logDto.AppResult != "" {
		s.logDto.AppResult = s.customLogger.logDto.AppResult
	} else {
		s.logDto.AppResult = "Success"
	}

	if data.Code != "" {
		s.logDto.AppResultCode = data.Code
	} else if s.customLogger.logDto.AppResultCode != "" {
		s.logDto.AppResultCode = s.customLogger.logDto.AppResultCode
	} else {
		s.logDto.AppResultCode = "20000"
	}

	if len(s.customLogger.summaryLogAdditionalInfo) > 0 {
		s.logDto.Sequences = append(s.logDto.Sequences, s.customLogger.summaryLogAdditionalInfo...)
		sequences := s.logDto.Sequences
		jsonBytes, err := json.Marshal(sequences)
		if err == nil {
			s.logDto.Messages = string(jsonBytes)
		}
		s.logDto.Sequences = nil
		s.customLogger.summaryLogAdditionalInfo = nil
	}

	s.clearNonSummaryLogParam()
	jsonBytes, err := json.Marshal(s.logDto)
	if err == nil {
		info := string(jsonBytes)
		s.logger.Info(info)
	}
}
