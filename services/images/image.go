// Package images creates service that is able to upload image
// and return list of images uploaded by user
package images

import (
	"errors"

	"github.com/jmoiron/sqlx"
)

// Storage interface defines storage methods needed by images service
type Storage interface {
	AddImage(img *Image) error
	LoadImages(images *[]Image, limit, offset, userID uint64) error
}

// DB type wraps *sqlx.DB for images-specific contex
type DB struct {
	*sqlx.DB
}

// Image describes image data that is stored in database
type Image struct {
	Filename string `json:"filename" db:"filename"`
	UserID   uint64 `json:"-" db:"user_id"`
}

// AddImage insert Image into database
func (db *DB) AddImage(img *Image) error {
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
	_, err = tx.Exec(
		`INSERT INTO images (filename, user_id) VALUES ($1, $2)`,
		img.Filename,
		img.UserID)
	if err != nil {
		err = tx.Rollback()
	} else {
		err = tx.Commit()
	}
	return err
}

// LoadImages selects Images from database
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
