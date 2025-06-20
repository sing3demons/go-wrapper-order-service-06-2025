package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

const (
	defaultMaxMemory = 32 << 20 // 32 MB
)

var (
	errNoFileFound    = errors.New("no files were bounded")
	errNonPointerBind = errors.New("bind error, cannot bind to a non pointer type")
	errNonSliceBind   = errors.New("bind error: input is not a pointer to a byte slice")
)

// Request is an abstraction over the underlying http.Request. This abstraction is useful because it allows us
// to create applications without being aware of the transport. cmd.Request is another such abstraction.
type Request struct {
	req        *http.Request
	pathParams map[string]string
}

const (
	// XSessionIdHeader is the header key for the session ID.
	XSessionIdHeader = "X-Session-Id"
	// XTransactionIdHeader is the header key for the transaction ID.
	XTransactionIdHeader = "X-Transaction-Id"
	// XTidHeader is the header key for the transaction ID (legacy).
	XTidHeader = "X-Tid"
)

// NewRequest creates a new GoFr Request instance from the given http.Request.
func NewRequest(r *http.Request) *Request {
	sessionId := r.Header.Get(XSessionIdHeader)
	if sessionId == "" {
		sessionId = uuid.New().String()
		r.Header.Set(XSessionIdHeader, sessionId)
	}
	transactionId := r.Header.Get(XTransactionIdHeader)
	if transactionId == "" {
		if r.Header.Get(XTidHeader) != "" {
			transactionId = r.Header.Get(XTidHeader)
		} else {
			transactionId = uuid.New().String()
		}
		r.Header.Set(XTransactionIdHeader, transactionId)
	}
	return &Request{
		req:        r,
		pathParams: mux.Vars(r),
	}
}

// Param returns the query parameter with the given key.
func (r *Request) Param(key string) map[string]string {
	return r.pathParams
}

// QueryParam returns the query parameter with the given key.
func (r *Request) QueryParam(key string) string {
	return r.req.URL.Query().Get(key)
}

// Context returns the context of the request.
func (r *Request) Context() context.Context {
	return r.req.Context()
}

func (r *Request) Header(key string) string {
	return r.req.Header.Get(key)
}

func (r *Request) Headers() map[string]any {
	headers := make(map[string]any)
	for key, values := range r.req.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		} else {
			headers[key] = nil
		}
	}
	return headers
}

func (r *Request) Method() string {
	return r.req.Method
}

func (r *Request) URL() string {
	if r.req.URL == nil {
		return ""
	}

	url := r.req.URL.String()
	if r.req.URL.RawQuery != "" {
		url += "?" + r.req.URL.RawQuery
	}

	return url
}


// PathParam retrieves a path parameter from the request.
func (r *Request) PathParam(key string) string {
	return r.pathParams[key]
}

// Bind parses the request body and binds it to the provided interface.
func (r *Request) Bind(i any) error {
	v := r.req.Header.Get("Content-Type")
	contentType := strings.Split(v, ";")[0]

	switch contentType {
	case "application/json":
		body, err := r.body()
		if err != nil {
			return err
		}

		return json.Unmarshal(body, &i)
	case "multipart/form-data":
		return r.bindMultipart(i)
	case "application/x-www-form-urlencoded":
		return r.bindFormURLEncoded(i)
	case "binary/octet-stream":
		return r.bindBinary(i)
	}

	return nil
}

// HostName retrieves the hostname from the request.
func (r *Request) HostName() string {
	proto := r.req.Header.Get("X-Forwarded-Proto")
	if proto == "" {
		proto = "http"
	}

	return fmt.Sprintf("%s://%s", proto, r.req.Host)
}

// Params returns a slice of strings containing the values associated with the given query parameter key.
// If the parameter is not present, an empty slice is returned.
func (r *Request) Params(key string) []string {
	values := r.req.URL.Query()[key]

	var result []string

	for _, value := range values {
		result = append(result, strings.Split(value, ",")...)
	}

	return result
}

func (r *Request) body() ([]byte, error) {
	bodyBytes, err := io.ReadAll(r.req.Body)
	if err != nil {
		return nil, err
	}

	r.req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	return bodyBytes, nil
}

func (r *Request) bindMultipart(ptr any) error {
	return r.bindForm(ptr, true)
}

func (r *Request) bindFormURLEncoded(ptr any) error {
	return r.bindForm(ptr, false)
}

func (r *Request) bindForm(ptr any, isMultipart bool) error {
	ptrVal := reflect.ValueOf(ptr)
	if ptrVal.Kind() != reflect.Ptr {
		return errNonPointerBind
	}

	ptrVal = ptrVal.Elem()

	var fd formData

	if isMultipart {
		if err := r.req.ParseMultipartForm(defaultMaxMemory); err != nil {
			return err
		}

		fd = formData{files: r.req.MultipartForm.File, fields: r.req.MultipartForm.Value}
	} else {
		if err := r.req.ParseForm(); err != nil {
			return err
		}

		fd = formData{fields: r.req.Form}
	}

	ok, err := fd.mapStruct(ptrVal, nil)
	if err != nil {
		return err
	}

	if !ok {
		if isMultipart {
			return errNoFileFound
		}

		return errFieldsNotSet
	}

	return nil
}

// bindBinary handles binding for binary/octet-stream content type.
func (r *Request) bindBinary(raw any) error {
	// Ensure raw is a pointer to a byte slice
	byteSlicePtr, ok := raw.(*[]byte)
	if !ok {
		return fmt.Errorf("%w: %v", errNonSliceBind, raw)
	}

	body, err := r.body()
	if err != nil {
		return fmt.Errorf("failed to read request body: %w", err)
	}

	// Assign the body to the provided slice
	*byteSlicePtr = body

	return nil
}

func (r *Request) SessionId() string {
	sessionId := r.req.Header.Get(XSessionIdHeader)
	if sessionId == "" {
		sessionId = uuid.New().String()
		r.req.Header.Set(XSessionIdHeader, sessionId)
	}

	return sessionId
}

func (r *Request) TransactionId() string {
	transactionId := r.req.Header.Get(XTransactionIdHeader)
	if transactionId == "" {
		if r.req.Header.Get(XTidHeader) != "" {
			transactionId = r.req.Header.Get(XTidHeader)
		} else {
			transactionId = uuid.New().String()
		}
		r.req.Header.Set(XTransactionIdHeader, transactionId)
	}

	return transactionId
}

func (r *Request) RequestId() string {
	requestId := r.req.Header.Get("X-Request-Id")
	if requestId == "" {
		requestId = uuid.New().String()
		r.req.Header.Set("X-Request-Id", requestId)
	}

	return requestId
}
