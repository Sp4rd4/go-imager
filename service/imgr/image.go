// Package imgr creates service that is able to upload image
// and return list of images uploaded by user.
package imgr

import (
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// Storage interface defines storage methods needed by images service.
type Storage interface {
	CreateImage(img *Image) error
	LoadImages(images *[]Image, limit, offset, userID uint64) error
}

// DB type wraps *sqlx.DB for images-specific context.
type DB struct {
	*sqlx.DB
}

// Image describes image data that is stored in database.
type Image struct {
	Filename string `json:"filename" db:"filename"`
	UserID   uint64 `json:"-" db:"user_id"`
}

// ErrUniqueIndexConflict is custom error for unique index conflicts
type ErrUniqueIndexConflict string

// Error is errors interface implementation for ErrUniqueIndexConflict
func (uic ErrUniqueIndexConflict) Error() string {
	return "Conflict on unique index in table " + string(uic)
}

// CreateImage insert Image into database.
func (db *DB) CreateImage(img *Image) error {
	if img == nil {
		return errors.New("image required")
	}
	if img.Filename == "" {
		return errors.New("image filename required")
	}
	tx, err := db.Beginx()
	if err != nil {
		return err
	}

	_, err = tx.Exec(`INSERT INTO images (filename, user_id) VALUES ($1, $2)`, img.Filename, img.UserID)
	handleConflictError(&err)

	if err != nil {
		if errT := tx.Rollback(); errT != nil {
			err = fmt.Errorf("First: %s, Second: %s", err, errT)
		}
	} else {
		err = tx.Commit()
	}
	return err
}

func handleConflictError(err *error) {
	if *err != nil {
		if pgerr, ok := (*err).(*pq.Error); ok && pgerr.Code == "23505" {
			*err = ErrUniqueIndexConflict(pgerr.Table)
		}
	}
}

// LoadImages selects Images from database.
func (db *DB) LoadImages(images *[]Image, limit, offset, userID uint64) error {
	qStr := `SELECT * FROM images WHERE user_id=$1 ORDER BY filename OFFSET $2`
	params := make([]interface{}, 2, 3)
	params[0] = userID
	params[1] = offset
	if limit > 0 {
		qStr += `LIMIT $3`
		params = append(params, limit)
	}

	err := db.Select(images, qStr, params...)
	return err
}
