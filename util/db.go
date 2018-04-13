package util

import (
	"errors"
	"os"

	"github.com/golang-migrate/migrate"
	"github.com/golang-migrate/migrate/database/postgres"
	// DB connection establishing is handled only in this file.
	_ "github.com/golang-migrate/migrate/source/file"
	"github.com/jmoiron/sqlx"
	// DB connection establishing is handled only in this file.
	_ "github.com/lib/pq"
)

// OpenDB opens connection to postgres DB
// and run migrations from given folder path if there are any not ran before.
func OpenDB(address, migrations string) (*sqlx.DB, error) {
	db, err := sqlx.Connect("postgres", address)
	if err != nil {
		return nil, err
	}

	driver, err := postgres.WithInstance(db.DB, &postgres.Config{})
	if err != nil {
		return db, err
	}

	if err = checkFolder(migrations); err != nil {
		return db, err
	}

	m, err := migrate.NewWithDatabaseInstance("file://"+migrations, "postgres", driver)
	if err != nil {
		return db, err
	}

	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		return db, err
	}

	return db, nil
}

func checkFolder(migrations string) error {
	fileInfo, err := os.Stat(migrations)
	if err != nil {
		return err
	}
	if !fileInfo.IsDir() {
		return errors.New("unable to load migrations folder")
	}
	return nil
}
