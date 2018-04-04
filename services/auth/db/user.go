package db

import (
	"errors"
)

type User struct {
	Login        string `json:"login" db:"login"`
	PasswordHash string `json:"-" db:"password_hash"`
	Id           int    `json:"id" db:"id"`
}

func (db *DB) CreateUser(login, passwordHash string) (*User, error) {
	if login == "" || passwordHash == "" {
		return nil, errors.New("User auth info required")
	}
	tx, err := db.Beginx()
	if err != nil {
		return nil, err
	}
	_, err = tx.Exec(
		`INSERT INTO users (login, password) VALUES ($1, $2)`,
		login,
		passwordHash)
	user := &User{}
	if err == nil {
		err = tx.Get(user, `SELECT id FROM users WHERE login=$1`, login)
	}
	if err != nil {
		tx.Rollback()
	} else {
		tx.Commit()
	}

	return nil, err
}

func (db *DB) LoadUserByLogin(login string) (*User, error) {
	if login == "" {
		return nil, nil
	}
	user := &User{}
	err := db.Get(user, `SELECT * FROM users WHERE login=$1`, login)
	if err != nil {
		return nil, err
	}

	if user.Id == 0 && user.Login == "" && user.PasswordHash == "" {
		user = nil
	}
	return user, err
}
