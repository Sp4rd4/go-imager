package imgr_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sp4rd4/go-imager/service/imgr"
	"github.com/sp4rd4/go-imager/util"
)

func TestDBCreateImage(t *testing.T) {
	db, err := util.OpenDB(os.Getenv("DATABASE_URL"), os.Getenv("MIGRATIONS_FOLDER"))
	if err != nil {
		t.Fatal(err)
	}
	defer util.CleanDB(t, db)
	imgDB := &imgr.DB{DB: db}

	for _, ex := range examplesDBCreateImage {
		t.Run(ex.name, func(t *testing.T) {
			defer cleanTable(t, imgDB)

			var err error
			for _, img := range ex.input {
				if errA := imgDB.CreateImage(img); errA != nil {
					err = errA
				}
			}
			assert.EqualValues(t, ex.wantErr, err, "Error should be as expected")
		})
	}
}

func TestDBLoadImages(t *testing.T) {
	db, err := util.OpenDB(os.Getenv("DATABASE_URL"), os.Getenv("MIGRATIONS_FOLDER"))
	if err != nil {
		t.Fatal(err)
	}
	defer util.CleanDB(t, db)
	imgDB := &imgr.DB{DB: db}

	for _, ex := range examplesDBLoadImages {
		for _, img := range ex.initial {
			err = imgDB.CreateImage(&img)
			if err != nil {
				t.Fatal(err)
			}
		}

		t.Run(ex.name, func(t *testing.T) {
			defer cleanTable(t, imgDB)

			imgs := make([]imgr.Image, 0)
			err := imgDB.LoadImages(&imgs, ex.limit, ex.offset, ex.userID)
			assert.EqualValues(t, ex.wantErr, err, "Error should be as expected")
			for i, img := range ex.want {
				if i < len(imgs) {
					assert.Equalf(t, img, imgs[i], "Loaded Image %d is not as expected", i)
				} else {
					t.Errorf("Image %d is absent", i)
				}
			}
		})
	}
}

func cleanTable(t *testing.T, db *imgr.DB) {
	if _, err := db.Exec(`TRUNCATE TABLE "images" CASCADE;`); err != nil {
		t.Fatal(err)
	}
}
