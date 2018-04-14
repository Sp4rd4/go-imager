// Package util contains various shared functionality between services.
package util

import (
	"io"
	"testing"

	"github.com/jmoiron/sqlx"
)

// added to generalize CloseAndCheck so it can use both *testing.T and *logrus.Logger
type fatalist interface {
	Fatal(args ...interface{})
}

// CloseAndCheck closes variable and notifies if error happens.
func CloseAndCheck(c io.Closer, reporter fatalist) {
	if c == nil {
		return
	}
	if err := c.Close(); err != nil {
		reporter.Fatal(err)
	}
}

// CleanDB removes all data and tables from db
func CleanDB(t *testing.T, db *sqlx.DB) {
	if _, err := db.Exec("DROP SCHEMA public CASCADE;CREATE SCHEMA public;"); err != nil {
		t.Fatal("Unable to clean db")
	}
	CloseAndCheck(db, t)
}
