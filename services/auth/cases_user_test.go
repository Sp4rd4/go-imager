package auth_test

import (
	"database/sql"
	"errors"

	"github.com/sp4rd4/go-imager/services/auth"
)

var examplesDBCreateUser = []struct {
	name    string
	input   []*auth.User
	wantErr error
}{
	{
		"Single valid",
		[]*auth.User{&auth.User{PasswordHash: "hash", Login: "login"}},
		nil,
	},
	{
		"Multiple valid",
		[]*auth.User{&auth.User{PasswordHash: "hash", Login: "login1"}, &auth.User{PasswordHash: "hash", Login: "login2"}},
		nil,
	},
	{
		"Duplicate user",
		[]*auth.User{&auth.User{PasswordHash: "hash", Login: "login"}, &auth.User{PasswordHash: "hash", Login: "login"}},
		auth.ErrUniqueIndexConflict("users"),
	},
	{
		"Missing user",
		[]*auth.User{nil},
		errors.New("user required"),
	},
	{
		"Missing login",
		[]*auth.User{&auth.User{PasswordHash: "hash"}},
		errors.New("user auth info required"),
	},
	{
		"Missing password_hash",
		[]*auth.User{&auth.User{Login: "login"}},
		errors.New("user auth info required"),
	},
}

var examplesDBLoadUserByLogin = []struct {
	name    string
	initial []*auth.User
	user    *auth.User
	want    *auth.User
	wantErr error
}{
	{
		"OK",
		[]*auth.User{
			&auth.User{PasswordHash: "hash", Login: "login1"},
			&auth.User{PasswordHash: "hash", Login: "login2"},
			&auth.User{PasswordHash: "hash", Login: "login3"},
		},
		&auth.User{Login: "login3"},
		&auth.User{PasswordHash: "hash", Login: "login3"},
		nil,
	},
	{
		"Missing login",
		[]*auth.User{
			&auth.User{PasswordHash: "hash", Login: "login1"},
			&auth.User{PasswordHash: "hash", Login: "login2"},
			&auth.User{PasswordHash: "hash", Login: "login3"},
		},
		&auth.User{Login: ""},
		&auth.User{Login: ""},
		sql.ErrNoRows,
	},
	{
		"Missing user",
		[]*auth.User{
			&auth.User{PasswordHash: "hash", Login: "login1"},
			&auth.User{PasswordHash: "hash", Login: "login2"},
			&auth.User{PasswordHash: "hash", Login: "login3"},
		},
		nil,
		nil,
		errors.New("user required"),
	},
	{
		"login not in db",
		[]*auth.User{
			&auth.User{PasswordHash: "hash", Login: "login1"},
			&auth.User{PasswordHash: "hash", Login: "login2"},
			&auth.User{PasswordHash: "hash", Login: "login3"},
		},
		&auth.User{Login: "login4"},
		&auth.User{Login: "login4"},
		sql.ErrNoRows,
	},
}
