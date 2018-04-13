// Package utils contains various shared functionality between services.
package utils

import (
	"io"
	"testing"

	"github.com/jmoiron/sqlx"
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
		t.Fatal(err)
	}
}

// CleanDB removes all data and tables from db
func CleanDB(t *testing.T, db *sqlx.DB) {
	if _, err := db.Exec("DROP SCHEMA public CASCADE;CREATE SCHEMA public;"); err != nil {
		t.Fatal("Unable to clean db")
	}
	CloseAndCheckTest(t, db)
}
