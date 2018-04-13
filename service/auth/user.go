package auth

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// Storage interface defines storage methods needed by auth service.
type Storage interface {
	CreateUser(user *User) error
	LoadUserByLogin(user *User) error
}

// DB type wraps *sqlx.DB for users-specific context.
type DB struct {
	*sqlx.DB
}

// User describes user data that is stored in database.
type User struct {
	Login        string `json:"login" db:"login"`
	PasswordHash string `json:"-" db:"password_hash"`
	ID           uint64 `json:"id" db:"id"`
}

// ErrUniqueIndexConflict is custom error for unique index conflicts
type ErrUniqueIndexConflict string

// Error is errors interface implementation for ErrUniqueIndexConflict
func (uic ErrUniqueIndexConflict) Error() string {
	return "Conflict on unique index in table " + string(uic)
}

// CreateUser creates passed user and updates his ID
func (db *DB) CreateUser(user *User) error {
	if user == nil {
		return errors.New("user required")
	}
	if user.Login == "" || user.PasswordHash == "" {
		return errors.New("user auth info required")
	}
	tx, err := db.Beginx()
	if err != nil {
		return err
	}

	err = insertAndLoadUser(tx, user)
	if err != nil {
		if errT := tx.Rollback(); errT != nil {
			err = fmt.Errorf("First: %s, Second: %s", err, errT)
		}
	} else {
		err = tx.Commit()
	}

	return err
}

func insertAndLoadUser(tx *sqlx.Tx, user *User) error {
	_, err := tx.Exec(`INSERT INTO users (login, password_hash) VALUES ($1, $2)`, user.Login, user.PasswordHash)
	if err != nil {
		if pgerr, ok := err.(*pq.Error); ok && pgerr.Code == "23505" {
			err = ErrUniqueIndexConflict(pgerr.Table)
		}
	}

	if err == nil {
		err = tx.Get(user, `SELECT id FROM users WHERE login=$1`, user.Login)
	}
	return err
}

// LoadUserByLogin loads user to passed var after looking up db record by login
func (db *DB) LoadUserByLogin(user *User) error {
	if user == nil {
		return errors.New("user required")
	}
	if user.Login == "" {
		return sql.ErrNoRows
	}
	err := db.Get(user, `SELECT * FROM users WHERE login=$1`, user.Login)
	return err
}
