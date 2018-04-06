package utils_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/sp4rd4/go-imager/utils"
	"github.com/stretchr/testify/assert"
)

func TestOpenDB(t *testing.T) {

	dbAddress := os.Getenv("DATABASE_URL")
	if dbAddress == "" {
		t.Fatalf("Need db link")
	}
	migrationsFolder, err := ioutil.TempDir("", "migrations")
	defer os.RemoveAll(migrationsFolder)

	assert := assert.New(t)
	db, err := utils.OpenDB("wrong", migrationsFolder)
	if assert.NotNil(err, "OpenDB should return error for incorrect db link") {
		assert.Nil(db, "OpenDB should return nil *sqlx.DB for incorrect db link")
	}

	db, err = utils.OpenDB(dbAddress, "migrationsFolder")
	if assert.NotNil(err, "OpenDB should return error for missing migrations folder") {
		assert.Nil(db, "OpenDB should return nil *sqlx.DB for missing migrations folder")
	}

	tmsp := time.Now().Unix()
	createMigration(t, migrationsFolder, "first", `CREATE TABLE "films" ("prod" varchar);`, tmsp)
	createMigration(t, migrationsFolder, "second", `CREATE TABLE "users" ("name" varchar);`, tmsp+1)
	db, err = utils.OpenDB(dbAddress, migrationsFolder)
	if assert.Nil(err, "OpenDB shouldn't return error with existing valid migrations") {
		if assert.NotNil(db, "OpenDB should return valid *sqlx.DB with existing invalid migrations") {
			_, err = db.Exec("DROP SCHEMA public CASCADE;CREATE SCHEMA public;")
			if err != nil {
				t.Fatalf("Unable to clean db after tests")
			}
			db.Close()
		}
	}

	createMigration(t, migrationsFolder, "first", `CREATE TABLE "films" ("prod" varchar);`, tmsp)
	createMigration(t, migrationsFolder, "second", `CREATE ms" ("prod");`, tmsp+1)
	db, err = utils.OpenDB(dbAddress, migrationsFolder)
	if assert.NotNil(err, "OpenDB should return error with existing invalid migrations") {
		assert.Nil(db, "OpenDB should return nil *sqlx.DB with existing invalid migrations")
	}

	db, err = sqlx.Connect("postgres", dbAddress)
	if err != nil {
		t.Fatalf("Unable to clean db after tests")
	}
	_, err = db.Exec("DROP SCHEMA public CASCADE;CREATE SCHEMA public;")
	if err != nil {
		t.Fatalf("Unable to clean db after tests")
	}
	db.Close()
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
