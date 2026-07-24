package response

import (
	"encoding/json"
	"net/http"
)

type Body struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func JSON(w http.ResponseWriter, status, code int, message string, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(Body{Code: code, Message: message, Data: data})
}

func OK(w http.ResponseWriter, data any) {
	JSON(w, http.StatusOK, 0, "ok", data)
}

func Fail(w http.ResponseWriter, status, code int, message string) {
	JSON(w, status, code, message, nil)
}
