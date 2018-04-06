package utils

import (
	"io"
	"net/http"
)

// JSONResponse generates json response with given body and status code, while setting proper content type
func JSONResponse(w http.ResponseWriter, status int, message string) {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(status)
	io.WriteString(w, message)
}
