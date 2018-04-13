package utils

import (
	"io"
	"net/http"

	log "github.com/sirupsen/logrus"
)

// JSONResponse generates json response with given body and status code, while setting proper content type.
func JSONResponse(w http.ResponseWriter, status int, message string, logger *log.Entry) {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err := io.WriteString(w, message); err != nil {
		logger.Error(err)
	}
}
