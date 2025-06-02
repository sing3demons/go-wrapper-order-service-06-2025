package commonlog

import (
	"encoding/json"
	"testing"

	"github.com/sing3demons/go-order-service/pkg/common-log/masking"
	"github.com/stretchr/testify/assert"
)

// test cloneAndMask

type InputData struct {
	Body any `json:"body"`
}

func TestCloneAndMask(t *testing.T) {
	var body = InputData{
		Body: map[string]any{
			"key1": "value1",
			"key2": "value2"},
	}
	var maskingData = []masking.MaskingOptionDto{{
		MaskingField: "body.key1",
		MaskingType:  masking.Full,
	}}

	data := cloneAndMask(body, maskingData, *masking.NewMaskingService())
	var result InputData
	jsonData, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("Failed to marshal data: %v", err)
	}
	err = json.Unmarshal(jsonData, &result)
	if err != nil {
		t.Fatalf("Failed to unmarshal data: %v", err)
	}

	actual := InputData{
		Body: map[string]any{
			"key1": "XXXXXX",
			"key2": "value2"},
	}

	// convert to interface{} for comparison

	assert.Equal(t, actual, result, "The masked value should be 'XXXXXX'")

}

func TestCloneAndMaskArray(t *testing.T) {
	body := map[string]interface{}{
		"body": []map[string]interface{}{
			{"key1": "value1"},
			{"key2": "value2"},
		},
	}

	maskingData := []masking.MaskingOptionDto{{
		MaskingField: "body.*.key1",
		MaskingType:  masking.Full,
		IsArray:      true,
	}, {
		MaskingField: "body.*.key2",
		MaskingType:  masking.Full,
		IsArray:      true,
	}}

	data := cloneAndMask(body, maskingData, *masking.NewMaskingService())

	expected := map[string]interface{}{
		"body": []interface{}{
			map[string]interface{}{"key1": "XXXXXX"},
			map[string]interface{}{"key2": "XXXXXX"},
		},
	}

	assert.Equal(t, expected, data, "The masked value should be 'XXXXXX'")
}
