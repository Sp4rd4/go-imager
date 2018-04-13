package images_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sp4rd4/go-imager/services/images"
	"github.com/sp4rd4/go-imager/utils"
)

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
				if errA := imgDB.AddImage(img); errA != nil {
					err = errA
				}
			}
			assert.EqualValues(t, ex.wantErr, err, "Error should be as expected")
			cleanTable(t, imgDB)
		})
	}
	utils.CleanDB(t, db)
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
	utils.CleanDB(t, db)
}

func cleanTable(t *testing.T, db *images.DB) {
	if _, err := db.Exec(`TRUNCATE TABLE "images" CASCADE;`); err != nil {
		t.Fatal(err)
	}
}
