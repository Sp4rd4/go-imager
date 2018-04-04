package db

import (
	"errors"
	"os"

	"github.com/golang-migrate/migrate"
	"github.com/golang-migrate/migrate/database/postgres"
	_ "github.com/golang-migrate/migrate/source/file"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type Storage interface {
	CreateUser(u *User) error
	LoadUserByLogin(login string) (*User, error)
}

type DB struct {
	*sqlx.DB
}

func Open(address, migrations string) (*DB, error) {
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
	m.Up()
	return &DB{db}, nil
}
