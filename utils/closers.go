// Package utils contains various shared functionality between services.
package utils

import (
	"io"
	"testing"

	log "github.com/sirupsen/logrus"
)

// CloseAndCheck closes variable and notifies if error happens.
func CloseAndCheck(c io.Closer, log *log.Logger) {
	if c == nil {
		return
	}
	if err := c.Close(); err != nil {
		log.Fatal(err)
	}
}

// CloseAndCheckTest closes variable and fails tests if error happens.
func CloseAndCheckTest(t *testing.T, c io.Closer) {
	if c == nil {
		return
	}
	if err := c.Close(); err != nil {
		log.Fatal(err)
	}
}
