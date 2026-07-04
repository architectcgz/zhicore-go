package httpapi

import (
	"encoding/json"
	"net/http"
	"time"
)

const (
	CodeSuccess = 200

	MessageSuccess = "操作成功"
)

type Response struct {
	Code      int    `json:"code"`
	Message   string `json:"message"`
	Data      any    `json:"data,omitempty"`
	Timestamp int64  `json:"timestamp"`
	TraceID   string `json:"traceId,omitempty"`
}

type ErrorDetail struct {
	// 对齐公开 `data.details[]` contract，表达路径级校验错误。
	Path       string `json:"path,omitempty"`
	Code       string `json:"code,omitempty"`
	MessageKey string `json:"messageKey,omitempty"`
}

type ErrorOption func(*Response)

type errorData struct {
	// 结构化错误细节放在 data 下，保持共享 envelope 的顶层字段稳定。
	Details []ErrorDetail `json:"details,omitempty"`
}

func WriteSuccess(w http.ResponseWriter, data any) {
	writeJSON(w, http.StatusOK, Response{
		Code:      CodeSuccess,
		Message:   MessageSuccess,
		Data:      data,
		Timestamp: time.Now().UnixMilli(),
	})
}

func WriteError(w http.ResponseWriter, status int, message string) {
	WriteErrorCode(w, status, status, message)
}

func WithTraceID(traceID string) ErrorOption {
	return func(response *Response) {
		if traceID == "" {
			return
		}

		response.TraceID = traceID
	}
}

func WithDetails(details []ErrorDetail) ErrorOption {
	return func(response *Response) {
		if len(details) == 0 {
			return
		}

		response.Data = errorData{Details: details}
	}
}

func WriteErrorCode(w http.ResponseWriter, status int, code int, message string, opts ...ErrorOption) {
	response := Response{
		Code:      code,
		Message:   message,
		Timestamp: time.Now().UnixMilli(),
	}

	for _, opt := range opts {
		if opt == nil {
			continue
		}

		opt(&response)
	}

	writeJSON(w, status, response)
}

func writeJSON(w http.ResponseWriter, status int, payload Response) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
