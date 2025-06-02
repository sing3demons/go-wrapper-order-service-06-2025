package product

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type ProductInfo struct {
	ProductId   string     `json:"id,omitempty"`
	Href        string     `json:"href,omitempty"`
	ProductName string     `json:"name"`
	Price       string     `json:"price"`
	Description string     `json:"description,omitempty"`
	CreatedAt   time.Time  `json:"createdAt,omitempty"`
	UpdatedAt   time.Time  `json:"updatedAt,omitempty"`
	DeletedAt   *time.Time `json:"deletedAt,omitempty"`
}

type ProcessMongoReq struct {
	Collection string `json:"collection"`
	Method     string `json:"method"`
	Query      any    `json:"query"`
	Document   any    `json:"document"`
	Options    any    `json:"options"`
}

func (r *ProcessMongoReq) RawString() string {
	method := strings.ToLower(r.Method)
	if strings.HasPrefix(method, "insert") {
		jsonDocumentBytes, _ := json.Marshal(r.Document)
		jsonDocument := strings.ReplaceAll(string(jsonDocumentBytes), `"`, "'")
		return fmt.Sprintf("%s.%s(%s)", r.Collection, r.Method, jsonDocument)
	}
	return "Collection: " + r.Collection + ", Method: " + r.Method + ", Query: " + r.Query.(string) + ", Document: " + r.Document.(string) + ", Options: " + r.Options.(string)
}
