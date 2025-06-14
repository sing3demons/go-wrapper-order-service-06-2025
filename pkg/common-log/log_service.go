package commonlog

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sing3demons/go-order-service/pkg/common-log/logAction"
	"github.com/sing3demons/go-order-service/pkg/common-log/masking"
)

type CustomLoggerService interface {
	Init(data LogDto)
	// getSummaryLogAdditionalInfo() any
	// setSummaryLogParameters(params SummaryParamsType)
	// getIsSetSummaryLogParameters() bool
	// getSummaryLogParameters() SummaryParamsType
	// setDependencyMetadata(metadata LogDependencyMetadata) CustomLoggerService
	GetLogDto() LogDto
	Update(key string, value any)
	Info(log logAction.LoggerAction, data any, options ...masking.MaskingOptionDto)
	Debug(log logAction.LoggerAction, data any, options ...masking.MaskingOptionDto)
	Error(log logAction.LoggerAction, data any, options ...masking.MaskingOptionDto)
	SetSummaryLogErrorSource(param ErrorSourceType) CustomLoggerService
	Flush()
	End(code int, message string)
	SetSummary(params LogEventTag) CustomLoggerService
	SetDependencyMetadata(metadata LogDependencyMetadata) CustomLoggerService

	// setSummaryLogAdditionalInfo(key string, value any) CustomLoggerService
	// setDetailLogAdditionalInfo(key string, value any) CustomLoggerService
}
type customLoggerService struct {
	logDto                    LogDto
	isSetSummaryLogParameters bool
	// summaryLogParameters     map[string]any
	additionalSummary        map[string]any
	summaryLogAdditionalInfo []Sequence

	detailLog      LoggerService
	summaryLog     LoggerService
	maskingService masking.MaskingService
	utilService    *Timer
}

var maskingService = masking.NewMaskingService()

type Timer struct {
	now   int64     // Unix timestamp in milliseconds
	begin time.Time // Duration since the start of the timer
}

func NewTimer() *Timer {
	return &Timer{
		now:   time.Now().UnixNano() / int64(time.Millisecond),
		begin: time.Now(),
	}
}

func NewLogger(detailLog LoggerService, summaryLog LoggerService, time *Timer) CustomLoggerService {
	return &customLoggerService{
		additionalSummary:         make(map[string]any),
		detailLog:                 detailLog,
		summaryLog:                summaryLog,
		maskingService:            *maskingService,
		isSetSummaryLogParameters: false,
		utilService:               time,
	}
}

func (c *customLoggerService) Init(data LogDto) {
	c.logDto = data
	c.logDto.LogType = "Detail"
	if c.logDto.SessionId == "" {
		c.logDto.SessionId = uuid.NewString()
	}

	if c.logDto.TransactionId == "" {
		c.logDto.TransactionId = uuid.NewString()
	}
	hostName, _ := os.Hostname()
	c.logDto.Instance = hostName
}

func (c *customLoggerService) GetLogDto() LogDto {
	return c.logDto
}

func (c *customLoggerService) Update(key string, value any) {
	v := reflect.ValueOf(&c.logDto).Elem()
	field := v.FieldByName(key)
	if field.IsValid() && field.CanSet() {
		val := reflect.ValueOf(value)
		if val.Type().AssignableTo(field.Type()) {
			field.Set(val)
		} else if val.Type().ConvertibleTo(field.Type()) {
			field.Set(val.Convert(field.Type()))
		}
	}
}

func (c *customLoggerService) SetDependencyMetadata(metadata LogDependencyMetadata) CustomLoggerService {
	if metadata.Dependency != "" {
		c.logDto.Dependency = metadata.Dependency
	}

	if metadata.ResponseTime != 0 {
		c.logDto.ResponseTime = metadata.ResponseTime
	}

	if (metadata.ResultCode) != "" {
		c.logDto.ResultCode = metadata.ResultCode
	}

	if metadata.ResultFlag != "" {
		c.logDto.ResultFlag = metadata.ResultFlag
	}
	return c
}

func (c *customLoggerService) Info(action logAction.LoggerAction, data any, options ...masking.MaskingOptionDto) {
	cloned := cloneAndMask(data, options, c.maskingService)
	c.logDto.Action = action.Action
	c.logDto.ActionDescription = action.ActionDescription
	c.logDto.SubAction = action.SubAction
	c.logDto.Message = toJSON(cloned)
	c.logDto.Timestamp = time.Now().Format(time.RFC3339) // c.utilService.SetTimestampFormat(time.Now())

	jsonBytes, err := json.MarshalIndent(c.logDto, "", "  ")
	if err != nil {
		c.detailLog.Errorf("Failed to marshal log data: %v", err)
		return
	}
	c.detailLog.Log(string(jsonBytes))
	c.logDto.SubAction = ""
}

func (c *customLoggerService) Debug(action logAction.LoggerAction, data any, options ...masking.MaskingOptionDto) {
	cloned := cloneAndMask(data, options, c.maskingService)
	c.logDto.Action = action.Action
	c.logDto.ActionDescription = action.ActionDescription
	c.logDto.SubAction = action.SubAction
	c.logDto.Message = toJSON(cloned)
	c.logDto.Timestamp = time.Now().Format(time.RFC3339)
	c.detailLog.Debug(c.logDto)
	c.logDto.SubAction = ""
}

func (c *customLoggerService) Error(action logAction.LoggerAction, data any, options ...masking.MaskingOptionDto) {
	cloned := cloneAndMask(data, options, c.maskingService)
	c.logDto.Action = action.Action
	c.logDto.ActionDescription = action.ActionDescription
	c.logDto.SubAction = action.SubAction
	c.logDto.Message = toJSON(cloned)
	c.logDto.Timestamp = time.Now().Format(time.RFC3339) // c.utilService.SetTimestampFormat(time.Now())
	jsonBytes, err := json.MarshalIndent(c.logDto, "", "  ")
	if err != nil {
		c.detailLog.Errorf("Failed to marshal log data: %v", err)
		return
	}
	c.detailLog.Error(string(jsonBytes))
	c.logDto.SubAction = ""
}

func (c *customLoggerService) SetSummaryLogErrorSource(param ErrorSourceType) CustomLoggerService {
	c.additionalSummary = map[string]any{
		"errorSource": map[string]any{
			"node":        param.Node,
			"code":        param.Code,
			"description": param.Description,
		},
	}
	return c
}
func (c *customLoggerService) SetSummary(param LogEventTag) CustomLoggerService {
	if c.summaryLogAdditionalInfo == nil {
		c.summaryLogAdditionalInfo = make([]Sequence, 0)
	}

	if param.Command == "" && c.logDto.ActionDescription != "" {
		param.Command = c.logDto.ActionDescription
	}

	sequenceResult := []SequenceResult{{
		Result:  param.Code,
		Desc:    param.Description,
		ResTime: param.ResTime,
	}}

	if len(c.summaryLogAdditionalInfo) > 0 {
		for i, seq := range c.summaryLogAdditionalInfo {
			if seq.Node == param.Node && seq.Command == param.Command {
				// Update existing sequence result
				seq.Result = append(seq.Result, SequenceResult{
					Result:  param.Code,
					Desc:    param.Description,
					ResTime: param.ResTime,
				})
				c.summaryLogAdditionalInfo[i] = seq
				return c
			}
		}
	}
	c.summaryLogAdditionalInfo = append(c.summaryLogAdditionalInfo, Sequence{
		Node:    param.Node,
		Command: param.Command,
		Result:  sequenceResult,
	})
	return c

}

func (c *customLoggerService) Flush() {
	summaryLog := NewSummaryLogService(c.summaryLog, c)
	summaryLog.Init(c.logDto)
	summaryLog.Flush()
	summaryLog = nil
	c.logDto = LogDto{} // Reset logDto after flushing
	c = nil
}

type resultCodeType struct {
	StatusCode string `json:"status"`
	ResultCode string `json:"resultCode"`
	Message    string `json:"message"`
}

func ConvertTTTTT(input string) string {
	return input + strings.Repeat("0", 5-len(input))

}

func mapHTTPStatusToSnakeCaseText(code int) string {
	text := http.StatusText(code)
	if text == "" {
		return "unknown"
	}
	return toSnakeCase(text)
}

// Converts "Not Found" → "not_found"
func toSnakeCase(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, "-", "_")
	return s
}

func expandResultCode(code int) resultCodeType {
	switch code {
	case http.StatusOK:
		return resultCodeType{
			StatusCode: strconv.Itoa(http.StatusOK),
			ResultCode: ConvertTTTTT(strconv.Itoa(code)),
			Message:    "Success",
		}
	case http.StatusBadRequest:
		return resultCodeType{
			StatusCode: strconv.Itoa(http.StatusBadRequest),
			ResultCode: ConvertTTTTT(strconv.Itoa(code)),
			Message:    mapHTTPStatusToSnakeCaseText(code),
		}
	case http.StatusUnauthorized:
		return resultCodeType{
			StatusCode: strconv.Itoa(http.StatusUnauthorized),
			ResultCode: ConvertTTTTT(strconv.Itoa(code)),
			Message:    mapHTTPStatusToSnakeCaseText(code),
		}
	case http.StatusForbidden:
		return resultCodeType{
			StatusCode: strconv.Itoa(http.StatusForbidden),
			ResultCode: ConvertTTTTT(strconv.Itoa(code)),
			Message:    mapHTTPStatusToSnakeCaseText(code),
		}
	case http.StatusNotFound:
		return resultCodeType{
			StatusCode: strconv.Itoa(http.StatusNotFound),
			ResultCode: ConvertTTTTT(strconv.Itoa(code)),
			Message:    mapHTTPStatusToSnakeCaseText(code),
		}
	case http.StatusInternalServerError:
		return resultCodeType{
			StatusCode: strconv.Itoa(http.StatusInternalServerError),
			ResultCode: ConvertTTTTT(strconv.Itoa(code)),
			Message:    mapHTTPStatusToSnakeCaseText(code),
		}
	case http.StatusServiceUnavailable:
		return resultCodeType{
			StatusCode: strconv.Itoa(http.StatusServiceUnavailable),
			ResultCode: ConvertTTTTT(strconv.Itoa(code)),
			Message:    mapHTTPStatusToSnakeCaseText(code),
		}
	case http.StatusGatewayTimeout:
		return resultCodeType{
			StatusCode: strconv.Itoa(http.StatusGatewayTimeout),
			ResultCode: ConvertTTTTT(strconv.Itoa(code)),
			Message:    mapHTTPStatusToSnakeCaseText(code),
		}
	case http.StatusConflict:
		return resultCodeType{
			StatusCode: strconv.Itoa(http.StatusConflict),
			ResultCode: ConvertTTTTT(strconv.Itoa(code)),
			Message:    mapHTTPStatusToSnakeCaseText(code),
		}
	case http.StatusTooManyRequests:
		return resultCodeType{
			StatusCode: strconv.Itoa(http.StatusTooManyRequests),
			ResultCode: ConvertTTTTT(strconv.Itoa(code)),
			Message:    mapHTTPStatusToSnakeCaseText(code),
		}
	case http.StatusNotImplemented:
		return resultCodeType{
			StatusCode: strconv.Itoa(http.StatusNotImplemented),
			ResultCode: ConvertTTTTT(strconv.Itoa(code)),
			Message:    mapHTTPStatusToSnakeCaseText(code),
		}
	default:
		return resultCodeType{
			StatusCode: "Error",
			ResultCode: ConvertTTTTT(strconv.Itoa(code)),
			Message:    mapHTTPStatusToSnakeCaseText(code),
		}
	}
}

func (c *customLoggerService) End(code int, message string) {
	result := expandResultCode(code)
	if message == "" {
		message = result.Message
	}
	stack := Stack{
		Code:    result.ResultCode,
		Message: message,
		Status:  result.StatusCode,
	}
	summaryLog := NewSummaryLogService(c.summaryLog, c)
	summaryLog.Init(c.logDto)
	summaryLog.End(stack)
	summaryLog = nil
	c.logDto = LogDto{} // Reset logDto after flushing
	c = nil
}

func isArrayOrSlice(data any) bool {
	kind := reflect.TypeOf(data).Kind()
	return kind == reflect.Slice || kind == reflect.Array
}

func cloneAndMask(data any, options []masking.MaskingOptionDto, masker masking.MaskingService) any {
	raw, err := json.Marshal(data)
	if err != nil {
		return data
	}

	// check is array
	if isArrayOrSlice(data) {
		var clones []map[string]any
		if err := json.Unmarshal(raw, &clones); err != nil {
			return data
		}

		for i := range clones {
			for _, opt := range options {
				if strings.Contains(opt.MaskingField, "*") {
					root := strings.TrimSuffix(strings.Split(opt.MaskingField, "*")[0], ".")
					lookupArr := GetObjectByStringKeys(clones[i], root)
					for index := range lookupArr {
						field := strings.Replace(opt.MaskingField, "*", strconv.Itoa(index), 1)
						SetNestedArrayProperty(clones[i], field, opt.MaskingType, masker)
					}
				} else {
					setNestedProperty(clones[i], opt.MaskingField, opt.MaskingType, masker)
				}
			}
		}

		return clones
	}

	var clone map[string]any
	if err := json.Unmarshal(raw, &clone); err != nil {
		return data
	}

	for _, opt := range options {
		if opt.IsArray && strings.Contains(opt.MaskingField, "*") {
			root := strings.TrimSuffix(strings.Split(opt.MaskingField, "*")[0], ".")
			suffix := strings.Split(opt.MaskingField, "*")[1]
			suffix = strings.TrimPrefix(suffix, ".")
			lookupArr := GetObjectByStringKeys(clone, root)
			for _, item := range lookupArr {
				elem, ok := item.(map[string]any)
				if !ok {
					continue
				}
				setNestedProperty(elem, suffix, opt.MaskingType, masker)
			}
			continue
		}
		setNestedProperty(clone, opt.MaskingField, opt.MaskingType, masker)
	}

	return clone
}

func SetNestedArrayProperty(obj map[string]any, path string, maskingType masking.MaskingType, masker masking.MaskingService) {
	keys := strings.Split(path, ".")
	var current any = obj

	for i := 0; i < len(keys)-1; i++ {
		key := keys[i]
		currentMap, ok := current.(map[string]any)
		if !ok {
			return
		}
		current = currentMap[key]
	}

	// Last key
	lastKey := keys[len(keys)-1]
	parentMap, ok := current.(map[string]any)
	if !ok {
		return
	}

	// Handle if parentMap[lastKey] is array of maps
	arr, ok := parentMap[lastKey].([]any)
	if !ok {
		return
	}
	for i, v := range arr {
		elem, ok := v.(map[string]any)
		if !ok {
			continue
		}
		oldVal, _ := elem["key1"].(string)
		elem["key1"] = masker.Masking(oldVal, maskingType)
		arr[i] = elem
	}
	parentMap[lastKey] = arr
}

func GetObjectByStringKeys(obj map[string]any, path string) []any {
	keys := strings.Split(path, ".")
	current := any(obj)
	for _, key := range keys {
		switch cur := current.(type) {
		case map[string]any:
			current = cur[key]
		case []any:
			// If the key is an index in brackets e.g. [0], handle it here (optional)
			return nil // or handle properly
		default:
			return nil
		}
		if current == nil {
			return nil
		}
	}
	if arr, ok := current.([]any); ok {
		return arr
	}
	return nil
}

func setNestedProperty(obj map[string]any, path string, maskType masking.MaskingType, masker masking.MaskingService) {
	keys := strings.Split(path, ".")
	current := obj
	for i := 0; i < len(keys)-1; i++ {
		if next, ok := current[keys[i]].(map[string]any); ok {
			current = next
		} else {
			return
		}
	}
	lastKey := keys[len(keys)-1]
	if val, ok := current[lastKey].(string); ok {
		current[lastKey] = masker.Masking(val, maskType)
	}
}

func toJSON(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case int, int64, float64, bool:
		return strings.TrimSpace(strings.ToLower(fmt.Sprintf("%v", val)))
	default:
		if v == nil {
			return ""
		}

		// b, _ := json.Marshal(v)
		jsonStr, _ := json.Marshal(v)
		strings := string(jsonStr)

		return strings
	}
}
