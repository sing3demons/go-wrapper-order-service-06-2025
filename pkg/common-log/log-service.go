package commonlog

import (
	"encoding/json"
	"fmt"
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
	// getLogDto() LogDto
	// update(key string, value any)
	Info(log logAction.LoggerAction, data any, options ...masking.MaskingOptionDto)
	Debug(log logAction.LoggerAction, data any, options ...masking.MaskingOptionDto)
	Error(log logAction.LoggerAction, data any, stack interface{}, options ...masking.MaskingOptionDto)
	SetSummaryLogErrorSource(param ErrorSourceType) CustomLoggerService
	Flush()
	AddSummary(params EventTag)
	SetDependencyMetadata(metadata LogDependencyMetadata) CustomLoggerService

	// setSummaryLogAdditionalInfo(key string, value any) CustomLoggerService
	// setDetailLogAdditionalInfo(key string, value any) CustomLoggerService
}
type customLoggerService struct {
	logDto LogDto
	// isSetSummaryLogParameters bool
	summaryLogParameters     map[string]interface{}
	additionalSummary        map[string]interface{}
	summaryLogAdditionalInfo []Sequence

	logger LoggerService
	// utilService    LoggerHelperService
	maskingService masking.MaskingService
}

var maskingService = masking.NewMaskingService()

func NewLogger(logger LoggerService) CustomLoggerService {
	return &customLoggerService{
		additionalSummary: make(map[string]interface{}),
		logger:            logger,
		maskingService:    *maskingService,
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

func (c *customLoggerService) Info(action logAction.LoggerAction, data interface{}, options ...masking.MaskingOptionDto) {
	cloned := cloneAndMask(data, options, c.maskingService)
	c.logDto.Action = action.Action
	c.logDto.ActionDescription = action.ActionDescription
	c.logDto.SubAction = action.SubAction
	c.logDto.Message = toJSON(cloned)
	c.logDto.Timestamp = time.Now().Format(time.RFC3339) // c.utilService.SetTimestampFormat(time.Now())

	jsonBytes, err := json.MarshalIndent(c.logDto, "", "  ")
	if err != nil {
		c.logger.Error("Failed to marshal log data", err)
		return
	}
	c.logger.Log(string(jsonBytes))
	// c.logDto = LogDto{}
	c.logDto.SubAction = ""
}

func (c *customLoggerService) Debug(action logAction.LoggerAction, data interface{}, options ...masking.MaskingOptionDto) {
	cloned := cloneAndMask(data, options, c.maskingService)
	c.logDto.Action = action.Action
	c.logDto.ActionDescription = action.ActionDescription
	c.logDto.SubAction = action.SubAction
	c.logDto.Message = toJSON(cloned)
	c.logDto.Timestamp = time.Now().Format(time.RFC3339)
	c.logger.Debug(c.logDto)
	c.logDto.SubAction = ""
}

func (c *customLoggerService) Error(action logAction.LoggerAction, data interface{}, stack interface{}, options ...masking.MaskingOptionDto) {
	cloned := cloneAndMask(data, options, c.maskingService)
	c.logDto.Action = action.Action
	c.logDto.ActionDescription = action.ActionDescription
	c.logDto.SubAction = action.SubAction
	c.logDto.Message = toJSON(cloned)
	c.logDto.Timestamp = time.Now().Format(time.RFC3339) // c.utilService.SetTimestampFormat(time.Now())
	c.logger.Error(c.logDto, stack)
	c.logDto.SubAction = ""
}

func (c *customLoggerService) SetSummaryLogErrorSource(param ErrorSourceType) CustomLoggerService {
	c.additionalSummary = map[string]interface{}{
		"errorSource": map[string]interface{}{
			"node":        param.Node,
			"code":        param.Code,
			"description": param.Description,
		},
	}
	return c
}
func (c *customLoggerService) AddSummary(param EventTag) {
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
				return
			}
		}
	}
	c.summaryLogAdditionalInfo = append(c.summaryLogAdditionalInfo, Sequence{
		Node:    param.Node,
		Command: param.Command,
		Result:  sequenceResult,
	})

}

func (c *customLoggerService) Flush() {
	summaryLog := NewSummaryLogService(c.logger.SummaryLog(), c)
	summaryLog.Init(c.logDto)
	summaryLog.Flush()
	summaryLog = nil
	c.logDto = LogDto{} // Reset logDto after flushing
	c = nil
}

func isArrayOrSlice(data interface{}) bool {
	kind := reflect.TypeOf(data).Kind()
	return kind == reflect.Slice || kind == reflect.Array
}

func cloneAndMask(data interface{}, options []masking.MaskingOptionDto, masker masking.MaskingService) interface{} {
	raw, err := json.Marshal(data)
	if err != nil {
		return data
	}

	// check is array
	if isArrayOrSlice(data) {
		var clones []map[string]interface{}
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

	var clone map[string]interface{}
	if err := json.Unmarshal(raw, &clone); err != nil {
		return data
	}

	for _, opt := range options {
		if opt.IsArray && strings.Contains(opt.MaskingField, "*") {
			root := strings.TrimSuffix(strings.Split(opt.MaskingField, "*")[0], ".")
			suffix := strings.Split(opt.MaskingField, "*")[1]
			suffix = strings.TrimPrefix(suffix, ".")
			lookupArr := GetObjectByStringKeys(clone, root)
			if lookupArr != nil {
				for _, item := range lookupArr {
					elem, ok := item.(map[string]interface{})
					if !ok {
						continue
					}
					setNestedProperty(elem, suffix, opt.MaskingType, masker)
				}
			}
			continue
		}
		setNestedProperty(clone, opt.MaskingField, opt.MaskingType, masker)
	}

	return clone
}

func SetNestedArrayProperty(obj map[string]interface{}, path string, maskingType masking.MaskingType, masker masking.MaskingService) {
	keys := strings.Split(path, ".")
	var current interface{} = obj

	for i := 0; i < len(keys)-1; i++ {
		key := keys[i]
		currentMap, ok := current.(map[string]interface{})
		if !ok {
			return
		}
		current = currentMap[key]
	}

	// Last key
	lastKey := keys[len(keys)-1]
	parentMap, ok := current.(map[string]interface{})
	if !ok {
		return
	}

	// Handle if parentMap[lastKey] is array of maps
	arr, ok := parentMap[lastKey].([]interface{})
	if !ok {
		return
	}
	for i, v := range arr {
		elem, ok := v.(map[string]interface{})
		if !ok {
			continue
		}
		oldVal, _ := elem["key1"].(string)
		elem["key1"] = masker.Masking(oldVal, maskingType)
		arr[i] = elem
	}
	parentMap[lastKey] = arr
}

func GetObjectByStringKeys(obj map[string]interface{}, path string) []interface{} {
	keys := strings.Split(path, ".")
	current := interface{}(obj)
	for _, key := range keys {
		switch cur := current.(type) {
		case map[string]interface{}:
			current = cur[key]
		case []interface{}:
			// If the key is an index in brackets e.g. [0], handle it here (optional)
			return nil // or handle properly
		default:
			return nil
		}
		if current == nil {
			return nil
		}
	}
	if arr, ok := current.([]interface{}); ok {
		return arr
	}
	return nil
}

func setNestedProperty(obj map[string]interface{}, path string, maskType masking.MaskingType, masker masking.MaskingService) {
	keys := strings.Split(path, ".")
	current := obj
	for i := 0; i < len(keys)-1; i++ {
		if next, ok := current[keys[i]].(map[string]interface{}); ok {
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
