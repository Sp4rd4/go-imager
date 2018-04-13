package images_test

import (
	"os"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"

	"github.com/sp4rd4/go-imager/services/images"
	"github.com/sp4rd4/go-imager/utils"
)

func cleanTable(t *testing.T, db *images.DB) {
	if _, err := db.Exec(`TRUNCATE TABLE "images" CASCADE;`); err != nil {
		t.Fatal(err)
	}
}

func TestDBAddImage(t *testing.T) {
	db, err := utils.OpenDB(os.Getenv("DATABASE_URL"), os.Getenv("MIGRATIONS_FOLDER"))
	if err != nil {
		t.Fatal(err)
	}
	imgDB := &images.DB{DB: db}
	for _, ex := range examplesDBAddImage {
		t.Run(ex.name, func(t *testing.T) {
			var err error
			for _, img := range ex.input {
				err = imgDB.AddImage(img)
			}
			assert.EqualValues(t, ex.wantErr, err, "Error should be as expected")
			cleanTable(t, imgDB)
		})
	}
	cleanDB(t, db)
}

func TestDBLoadImages(t *testing.T) {
	db, err := utils.OpenDB(os.Getenv("DATABASE_URL"), os.Getenv("MIGRATIONS_FOLDER"))
	if err != nil {
		t.Fatal(err)
	}
	imgDB := &images.DB{DB: db}
	for _, ex := range examplesDBLoadImages {
		t.Run(ex.name, func(t *testing.T) {
			for _, img := range ex.initial {
				err = imgDB.AddImage(&img)
				if err != nil {
					t.Fatal(err)
				}
			}
			imgs := make([]images.Image, 0)
			err := imgDB.LoadImages(&imgs, ex.limit, ex.offset, ex.userID)
			assert.EqualValues(t, ex.wantErr, err, "Error should be as expected")
			for i, img := range ex.want {
				if i < len(imgs) {
					assert.Equalf(t, img, imgs[i], "Loaded Image %d is not as expected", i)
				} else {
					t.Errorf("Image %d is absent", i)
				}
			}
			cleanTable(t, imgDB)
		})
	}
	cleanDB(t, db)
}

func cleanDB(t *testing.T, db *sqlx.DB) {
	if _, err := db.Exec("DROP SCHEMA public CASCADE;CREATE SCHEMA public;"); err != nil {
		t.Fatal("Unable to clean db before tests")
	}
	utils.CloseAndCheckTest(t, db)
}
