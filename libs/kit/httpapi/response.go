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

func WriteSuccess(w http.ResponseWriter, data any) {
	writeJSON(w, http.StatusOK, Response{
		Code:      CodeSuccess,
		Message:   MessageSuccess,
		Data:      data,
		Timestamp: time.Now().UnixMilli(),
	})
}

func WriteError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, Response{
		Code:      status,
		Message:   message,
		Timestamp: time.Now().UnixMilli(),
	})
}

func writeJSON(w http.ResponseWriter, status int, payload Response) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
