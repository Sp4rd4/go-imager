package utils

import (
	"errors"
	"os"

	"github.com/golang-migrate/migrate"
	"github.com/golang-migrate/migrate/database/postgres"
	_ "github.com/golang-migrate/migrate/source/file"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func OpenDB(address, migrations string) (*sqlx.DB, error) {
	db, err := sqlx.Connect("postgres", address)
	if err != nil {
		return nil, err
	}
	driver, err := postgres.WithInstance(db.DB, &postgres.Config{})
	fileInfo, err := os.Stat(migrations)
	if err != nil {
		return nil, err
	}
	if !fileInfo.IsDir() {
		return nil, errors.New("Unable to load migrationsfolder")
	}
	if err != nil {
		return nil, errors.New("Unable to access migrations folder")
	}
	m, err := migrate.NewWithDatabaseInstance("file://"+migrations, "postgres", driver)
	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		return nil, err
	}
	return db, nil
}
