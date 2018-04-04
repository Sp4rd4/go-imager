package utils

import (
	"io"
	"net/http"
)

func JsonResponse(w http.ResponseWriter, status int, message string) {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(status)
	io.WriteString(w, message)
}
