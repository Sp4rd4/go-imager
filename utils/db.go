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
	if err != nil {
		db.Close()
		return nil, err
	}

	fileInfo, err := os.Stat(migrations)
	if err != nil {
		db.Close()
		return nil, err
	}
	if !fileInfo.IsDir() {
		db.Close()
		return nil, errors.New("Unable to load migrationsfolder")
	}

	m, err := migrate.NewWithDatabaseInstance("file://"+migrations, "postgres", driver)
	if err != nil {
		db.Close()
		return nil, err
	}

	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		db.Close()
		return nil, err
	}

	return db, nil
}
