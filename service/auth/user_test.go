package auth_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/sp4rd4/go-imager/service/auth"
	"github.com/sp4rd4/go-imager/util"
	"github.com/stretchr/testify/assert"
)

func TestDBCreateUser(t *testing.T) {
	db, err := util.OpenDB(os.Getenv("DATABASE_URL"), os.Getenv("MIGRATIONS_FOLDER"))
	if err != nil {
		t.Fatal(err)
	}
	atDB := &auth.DB{DB: db}
	defer util.CleanDB(t, db)

	for _, ex := range examplesDBCreateUser {
		defer cleanTable(t, atDB)

		t.Run(ex.name, func(t *testing.T) {
			var err error
			expectedUsers := make([]*auth.User, len(ex.input))
			for i, usr := range ex.input {
				usrN := usr
				expectedUsers[i] = usrN
				if errA := atDB.CreateUser(usr); errA != nil {
					err = errA
				}
			}
			assert.EqualValues(t, ex.wantErr, err, "Error should be as expected")
			if err == nil {
				for i, usr := range ex.input {
					if usr != nil && expectedUsers[i] != nil {
						assert.Equal(t, expectedUsers[i].Login, usr.Login, "User login should be as expected")
						assert.Equal(t, expectedUsers[i].PasswordHash, usr.PasswordHash, "User password hash should be as expected")
						assert.NotZero(t, usr.ID, "User password hash should be as expected")
						return
					}
					assert.Equal(t, expectedUsers[i], usr, "User should be as expected")
				}
			}
		})
	}
}

func TestDBLoadUserByLogin(t *testing.T) {
	db, err := util.OpenDB(os.Getenv("DATABASE_URL"), os.Getenv("MIGRATIONS_FOLDER"))
	if err != nil {
		t.Fatal(err)
	}
	atDB := &auth.DB{DB: db}
	defer util.CleanDB(t, db)

	for _, ex := range examplesDBLoadUserByLogin {
		for _, usr := range ex.initial {
			if err := atDB.CreateUser(usr); err != nil {
				fmt.Println(ex)
				t.Fatal(err)
			}
		}

		t.Run(ex.name, func(t *testing.T) {
			defer cleanTable(t, atDB)

			err := atDB.LoadUserByLogin(ex.user)
			assert.Equal(t, ex.wantErr, err, "Error should be as expected")
			if err == nil && ex.user != nil && ex.want != nil {
				assert.Equal(t, ex.want.Login, ex.user.Login, "User login should be as expected")
				assert.Equal(t, ex.want.PasswordHash, ex.user.PasswordHash, "User password hash should be as expected")
				assert.NotZero(t, ex.user.ID, "User password hash should be as expected")
				return
			}
			assert.Equal(t, ex.want, ex.user, "User should be as expected")
		})
	}
}

func cleanTable(t *testing.T, db *auth.DB) {
	if _, err := db.Exec(`TRUNCATE TABLE "users" CASCADE;`); err != nil {
		t.Fatal(err)
	}
}
