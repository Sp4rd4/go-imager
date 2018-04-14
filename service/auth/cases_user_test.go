package auth_test

import (
	"database/sql"
	"errors"

	"github.com/sp4rd4/go-imager/service/auth"
)

var examplesDBCreateUser = []struct {
	name    string
	input   []*auth.User
	wantErr error
}{
	{
		name:    "Single valid",
		input:   []*auth.User{{PasswordHash: "hash", Login: "login"}},
		wantErr: nil,
	},
	{
		name:    "Multiple valid",
		input:   []*auth.User{{PasswordHash: "hash", Login: "login1"}, {PasswordHash: "hash", Login: "login2"}},
		wantErr: nil,
	},
	{
		name:    "Duplicate user",
		input:   []*auth.User{{PasswordHash: "hash", Login: "login"}, {PasswordHash: "hash", Login: "login"}},
		wantErr: auth.ErrUniqueIndexConflict("users"),
	},
	{
		name:    "Missing user",
		input:   []*auth.User{nil},
		wantErr: errors.New("user required"),
	},
	{
		name:    "Missing login",
		input:   []*auth.User{{PasswordHash: "hash"}},
		wantErr: errors.New("user fields required"),
	},
	{
		name:    "Missing password_hash",
		input:   []*auth.User{{Login: "login"}},
		wantErr: errors.New("user fields required"),
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
		name: "OK",
		initial: []*auth.User{
			{PasswordHash: "hash", Login: "login1"},
			{PasswordHash: "hash", Login: "login2"},
			{PasswordHash: "hash", Login: "login3"},
		},
		user:    &auth.User{Login: "login3"},
		want:    &auth.User{PasswordHash: "hash", Login: "login3"},
		wantErr: nil,
	},
	{
		name: "Missing login",
		initial: []*auth.User{
			{PasswordHash: "hash", Login: "login1"},
			{PasswordHash: "hash", Login: "login2"},
			{PasswordHash: "hash", Login: "login3"},
		},
		user:    &auth.User{Login: ""},
		want:    &auth.User{Login: ""},
		wantErr: sql.ErrNoRows,
	},
	{
		name: "Missing user",
		initial: []*auth.User{
			{PasswordHash: "hash", Login: "login1"},
			{PasswordHash: "hash", Login: "login2"},
			{PasswordHash: "hash", Login: "login3"},
		},
		user:    nil,
		want:    nil,
		wantErr: errors.New("user required"),
	},
	{
		name: "login not in db",
		initial: []*auth.User{
			{PasswordHash: "hash", Login: "login1"},
			{PasswordHash: "hash", Login: "login2"},
			{PasswordHash: "hash", Login: "login3"},
		},
		user:    &auth.User{Login: "login4"},
		want:    &auth.User{Login: "login4"},
		wantErr: sql.ErrNoRows,
	},
}
