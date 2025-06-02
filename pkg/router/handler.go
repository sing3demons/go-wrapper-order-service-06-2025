package router

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"go.opentelemetry.io/otel/trace"

	config "github.com/sing3demons/go-order-service/configs"
	commonlog "github.com/sing3demons/go-order-service/pkg/common-log"
	gokpHTTP "github.com/sing3demons/go-order-service/pkg/http"
	kafkaService "github.com/sing3demons/go-order-service/pkg/kafka"
)

const colorCodeError = 202 // 202 is red color code

type Handler func(c *Context) error

/*
Developer Note: There is an implementation where we do not need this internal handler struct
and directly use Handler. However, in that case the container dependency is not injected and
has to be created inside ServeHTTP method, which will result in multiple unnecessary calls.
This is what we implemented first.

There is another possibility where we write our own Router implementation and let httpServer
use that router which will return a Handler and httpServer will then create the context with
injecting container and call that Handler with the new context. A similar implementation is
done in CMD. Since this will require us to write our own router - we are not taking that path
for now. In the future, this can be considered as well if we are writing our own HTTP router.
*/

type handler struct {
	function       Handler
	requestTimeout time.Duration
	kafkaService.KafkaClient
	Logger commonlog.LoggerService
	conf   *config.Config
}

type ErrorLogEntry struct {
	TraceID string `json:"trace_id,omitempty"`
	Error   string `json:"error,omitempty"`
}

func (el *ErrorLogEntry) PrettyPrint(writer io.Writer) {
	fmt.Fprintf(writer, "\u001B[38;5;8m%s \u001B[38;5;%dm%s \n", el.TraceID, colorCodeError, el.Error)
}

func (h handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c := newContext(w, gokpHTTP.NewRequest(r), h.KafkaClient, h.Logger, h.conf)
	traceID := trace.SpanFromContext(r.Context()).SpanContext().TraceID().String()

	if websocket.IsWebSocketUpgrade(r) {
		// If the request is a WebSocket upgrade, do not apply the timeout
		c.Context = r.Context()
	} else if h.requestTimeout != 0 {
		ctx, cancel := context.WithTimeout(r.Context(), h.requestTimeout)
		defer cancel()

		c.Context = ctx
	}

	done := make(chan struct{})
	panicked := make(chan struct{})

	var (
		err error
	)

	go func() {
		defer func() {
			panicRecoveryHandler(recover(), panicked)
		}()
		// Execute the handler function
		err = h.function(c)
		h.logError(traceID, err)
		close(done)
	}()

	select {
	case <-c.Context.Done():
		// If the context's deadline has been exceeded, return a timeout error response
		if errors.Is(c.Err(), context.DeadlineExceeded) {
			err = errors.New("request timed out")
		}
	case <-done:
		handleWebSocketUpgrade(r)
	case <-panicked:
		err = errors.New("internal server error")
	}

	// Handler function completed
	if err != nil {
		c.ResponseWriter.Write([]byte(err.Error()))
		return
	}

}

func liveHandler(*Context) (any, error) {
	return struct {
		Status string `json:"status"`
	}{Status: "UP"}, nil
}

func catchAllHandler(*Context) (any, error) {
	return nil, nil
}

func panicRecoveryHandler(re any, panicked chan struct{}) {
	if re == nil {
		return
	}

	close(panicked)
	// log.Error(panicLog{
	// 	Error:      fmt.Sprint(re),
	// 	StackTrace: string(debug.Stack()),
	// })
}

// Log the error(if any) with traceID and errorMessage.
func (h handler) logError(traceID string, err error) {
	if err != nil {
		errorLog := &ErrorLogEntry{TraceID: traceID, Error: err.Error()}
		fmt.Println(errorLog)

	}
}

func handleWebSocketUpgrade(r *http.Request) {
	if websocket.IsWebSocketUpgrade(r) {
		// Do not respond with HTTP headers since this is a WebSocket request
		return
	}
}
