package images_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sp4rd4/go-imager/services/images"
	"github.com/sp4rd4/go-imager/utils"
)

func setupDB(t *testing.T) *images.DB {
	db, err := utils.OpenDB(os.Getenv("DATABASE_URL"), os.Getenv("MIGRATIONS_FOLDER"))
	if err != nil {
		t.Fatal(err)
	}
	return &images.DB{DB: db}
}

func cleanTable(t *testing.T, db *images.DB) {
	if _, err := db.Exec(`TRUNCATE TABLE "images" CASCADE;`); err != nil {
		t.Fatal(err)
	}
}

func TestInsertImage(t *testing.T) {
	db := setupDB(t)
	defer cleanTable(t, db)
	img := images.Image{
		Filename: "test",
		UserID:   1,
	}
	err := db.InsertImage(&img)
	assert := assert.New(t)
	if assert.Nil(err) {
		var count int
		db.Get(&count, `SELECT count(*) FROM images;`)
		assert.Equal(1, count)
		var imgActual images.Image
		db.Get(&imgActual, `SELECT * FROM images LIMIT 1;`)
		assert.Equal(img, imgActual)
	}
}

func TestInsertIncorrectImage(t *testing.T) {
	db := setupDB(t)
	defer cleanTable(t, db)
	img := images.Image{
		Filename: "",
		UserID:   1,
	}
	err := db.InsertImage(&img)
	assert := assert.New(t)
	if assert.NotNil(err) {
		assert.Equal("Image required", err.Error)
	}
}
