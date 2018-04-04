package images

import (
	"errors"

	"github.com/jmoiron/sqlx"
)

type Storage interface {
	InsertImage(img *Image) error
	SelectImages(images *[]Image, limit, offset int, userId uint64) error
}

type DB struct {
	*sqlx.DB
}

type Image struct {
	Filename string `json:"filename" db:"filename"`
	UserId   uint64 `json:"-" db:"user_id"`
}

func (db *DB) InsertImage(img *Image) error {
	if img == nil {
		return errors.New("Image required")
	} else if img.Filename == "" {
		return errors.New("Image filename required")
	}
	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	_, err = tx.Exec(
		`INSERT INTO images (filename, user_id) VALUES ($1, $2)`,
		img.Filename,
		img.UserId)
	if err != nil {
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return err
}

func (db *DB) SelectImages(images *[]Image, limit, offset int, userId uint64) error {
	qStr := `SELECT * FROM images WHERE user_id=$1 ORDER BY filename OFFSET $2`
	params := make([]interface{}, 2, 3)
	params[0] = userId
	params[1] = offset
	if limit > 0 {
		qStr += `LIMIT $3`
		params = append(params, limit)
	}

	err := db.Select(images, qStr, params...)
	if err != nil {
		return err
	}
	return nil
}
