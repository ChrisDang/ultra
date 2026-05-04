package response

import (
	"encoding/json"
	"net/http"
)

type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ErrorInfo  `json:"error,omitempty"`
}

type ErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func JSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(Response{Success: true, Data: data})
}

func Error(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(Response{
		Success: false,
		Error:   &ErrorInfo{Code: code, Message: message},
	})
}

func BadRequest(w http.ResponseWriter, msg string)   { Error(w, 400, "BAD_REQUEST", msg) }
func Unauthorized(w http.ResponseWriter, msg string)  { Error(w, 401, "UNAUTHORIZED", msg) }
func Forbidden(w http.ResponseWriter, msg string)     { Error(w, 403, "FORBIDDEN", msg) }
func NotFound(w http.ResponseWriter, msg string)      { Error(w, 404, "NOT_FOUND", msg) }
func Conflict(w http.ResponseWriter, msg string)      { Error(w, 409, "CONFLICT", msg) }
func InternalError(w http.ResponseWriter, msg string) { Error(w, 500, "INTERNAL_ERROR", msg) }
