package auth

import (
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"
)

type Storage interface {
	CreateUser(user *User) error
	LoadUserByLogin(user *User) error
}

type DB struct {
	*sqlx.DB
}

type User struct {
	Login        string `json:"login" db:"login"`
	PasswordHash string `json:"-" db:"password_hash"`
	ID           uint64 `json:"id" db:"id"`
}

func (db *DB) CreateUser(user *User) error {
	if user.Login == "" || user.PasswordHash == "" {
		return errors.New("user auth info required")
	}
	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	_, err = tx.Exec(
		`INSERT INTO users (login, password_hash) VALUES ($1, $2)`,
		user.Login,
		user.PasswordHash)
	if err == nil {
		err = tx.Get(user, `SELECT id FROM users WHERE login=$1`, user.Login)
	}
	if err != nil {
		err = tx.Rollback()
	} else {
		err = tx.Commit()
	}

	return err
}

func (db *DB) LoadUserByLogin(user *User) error {
	if user.Login == "" {
		return sql.ErrNoRows
	}
	err := db.Get(user, `SELECT * FROM users WHERE login=$1`, user.Login)
	return err
}
