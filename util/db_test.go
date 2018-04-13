package util_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/sp4rd4/go-imager/util"
	"github.com/stretchr/testify/assert"
)

func TestOpenDB(t *testing.T) {

	dbAddress := os.Getenv("DATABASE_URL")
	if dbAddress == "" {
		t.Fatal("Need db link")
	}
	db, err := sqlx.Connect("postgres", dbAddress)
	if err != nil {
		t.Fatal(err)
	}
	util.CleanDB(t, db)

	subTestInvalidURL(t)
	subTestInvalidMigrationFolder(t, dbAddress)
	subTestValidMigrations(t, dbAddress)
	subTestInvalidMigrations(t, dbAddress)
}

func subTestInvalidURL(t *testing.T) {
	migrationsFolder, err := ioutil.TempDir("", "migrations")
	if err != nil {
		t.Fatal("Unable to create temp dir")
	}
	defer os.RemoveAll(migrationsFolder)

	t.Run("Invalid DB URL", func(t *testing.T) {
		db, err := util.OpenDB("wrong", migrationsFolder)
		if assert.NotNil(t, err, "OpenDB should return error for incorrect db link") {
			assert.Nil(t, db, "OpenDB should return nil *sqlx.DB for incorrect db link")
		}
	})
}

func subTestInvalidMigrationFolder(t *testing.T, dbAddress string) {
	t.Run("Invalid Migrations Folder", func(t *testing.T) {
		db, err := util.OpenDB(dbAddress, "migrationsFolder")
		if assert.NotNil(t, err, "OpenDB should return error for missing migrations folder") {
			if assert.NotNil(t, db, "OpenDB shouldn't return nil *sqlx.DB for missing migrations folder") {
				util.CloseAndCheck(db, t)
			}
		}
	})
}

func subTestValidMigrations(t *testing.T, dbAddress string) {
	migrationsFolder, err := ioutil.TempDir("", "migrations")
	if err != nil {
		t.Fatal("Unable to create temp dir")
	}
	defer os.RemoveAll(migrationsFolder)

	tmsp := time.Now().Unix()
	createMigration(t, migrationsFolder, "first", `CREATE TABLE "films" ("prod" varchar);`, tmsp)
	createMigration(t, migrationsFolder, "second", `CREATE TABLE "users" ("name" varchar);`, tmsp+1)

	t.Run("ValidMigrations", func(t *testing.T) {
		db, err := util.OpenDB(dbAddress, migrationsFolder)
		if assert.Nil(t, err, "OpenDB shouldn't return error with existing valid migrations") {
			if assert.NotNil(t, db, "OpenDB should return valid *sqlx.DB with existing invalid migrations") {
				util.CleanDB(t, db)
			}
		}
	})
}

func subTestInvalidMigrations(t *testing.T, dbAddress string) {
	migrationsFolder, err := ioutil.TempDir("", "migrations")
	if err != nil {
		t.Fatal("Unable to create temp dir")
	}
	defer os.RemoveAll(migrationsFolder)

	tmsp := time.Now().Unix()
	createMigration(t, migrationsFolder, "first", `CREATE TABLE "films" ("prod" varchar);`, tmsp)
	createMigration(t, migrationsFolder, "second", `CREATE ms" ("prod");`, tmsp+1)

	t.Run("Invalid existing migrations", func(t *testing.T) {
		db, err := util.OpenDB(dbAddress, migrationsFolder)
		if assert.NotNil(t, err, "OpenDB should return error with existing invalid migrations") {
			if assert.NotNil(t, db, "OpenDB shouldn't return nil *sqlx.DB with existing invalid migrations") {
				util.CleanDB(t, db)
			}
		}
	})
}

func createMigration(t *testing.T, dir, name, sql string, timestamp int64) {
	base := fmt.Sprintf("%v/%v_%v.", dir, timestamp, name)
	createFile(t, base+"up.sql", sql)
	createFile(t, base+"down.sql", "SELECT 1;")
}

func createFile(t *testing.T, name, content string) {
	if err := ioutil.WriteFile(name, []byte(content), 0666); err != nil {
		t.Fatal(err)
	}
}
